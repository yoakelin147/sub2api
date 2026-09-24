package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type supplierPasswordEncryptor struct{}

func (supplierPasswordEncryptor) Encrypt(value string) (string, error) {
	return "encrypted:" + value, nil
}
func (supplierPasswordEncryptor) Decrypt(value string) (string, error) {
	return strings.TrimPrefix(value, "encrypted:"), nil
}

func TestSupplierOAuthLoginPasswordEncryptedAndScoped(t *testing.T) {
	proxyID, supplierID := int64(11), int64(7)
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{
		ID: supplierID, Status: domain.SupplierStatusActive, ReviewRequired: true,
		AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeOAuth}},
	}}
	accountRepo := &supplierAccountRepositoryStub{}
	cfg := &config.Config{}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), nil, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), cfg, supplierPasswordEncryptor{})
	input := CreateSupplierAccountInput{Name: "OAuth", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "token", "email": "login@example.com", "password": "web-secret"}, ProxyID: &proxyID}
	_, err := svc.Create(context.Background(), supplierID, input)
	require.ErrorIs(t, err, ErrSupplierPasswordEncryptionRequired)
	require.Nil(t, accountRepo.created)
	cfg.Totp.EncryptionKeyConfigured = true
	account, err := svc.Create(context.Background(), supplierID, input)
	require.NoError(t, err)
	require.Equal(t, "encrypted:web-secret", account.Credentials["login_password_encrypted"])
	require.NotContains(t, account.Credentials, "password")
	accountRepo.account = account
	password, err := svc.RevealLoginPassword(context.Background(), supplierID, account.ID)
	require.NoError(t, err)
	require.Equal(t, "web-secret", password)
	_, err = svc.RevealLoginPassword(context.Background(), supplierID+1, account.ID)
	require.Error(t, err)
}

func TestSupplierOAuthRequiresWebLoginFields(t *testing.T) {
	_, err := ValidateSupplierAccountCredentials(&config.Config{}, PlatformOpenAI, AccountTypeOAuth, map[string]any{"access_token": "token"})
	require.ErrorIs(t, err, ErrSupplierCredentialsInvalid)
	_, err = ValidateSupplierAccountCredentials(&config.Config{}, PlatformOpenAI, AccountTypeAPIKey, map[string]any{"api_key": "token", "password": "secret"})
	require.ErrorIs(t, err, ErrSupplierCredentialsInvalid)
}

func TestSupplierAccountServiceCreateUsesSafePendingDefaults(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{
		ID: 7, Status: domain.SupplierStatusActive, ReviewRequired: true,
		AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}},
	}}
	accountRepo := &supplierAccountRepositoryStub{}
	proxyID := int64(11)
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), nil, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)

	account, err := svc.Create(context.Background(), 7, CreateSupplierAccountInput{
		ExternalID: " vendor-1 ", Name: " Primary ", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": " secret ", "base_url": "https://api.openai.com/"},
		ProxyID:     &proxyID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(7), *account.SupplierID)
	require.Equal(t, "vendor-1", *account.SupplierExternalID)
	require.Equal(t, AccountReviewStatusPending, account.ReviewStatus)
	require.Equal(t, StatusDisabled, account.Status)
	require.False(t, account.Schedulable)
	require.Equal(t, "secret", account.Credentials["api_key"])
	require.Equal(t, &proxyID, account.ProxyID)
}

func TestSupplierAccountServiceCreateRejectsKindNotAllowedForSupplier(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive}}
	svc := NewSupplierAccountService(&supplierAccountRepositoryStub{}, NewSupplierService(supplierRepo), nil, nil, &config.Config{}, nil)

	_, err := svc.Create(context.Background(), 7, CreateSupplierAccountInput{
		Name: "Primary", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "secret"},
	})
	require.ErrorIs(t, err, ErrSupplierAccountKindNotAllowed)
}

func TestSupplierAccountServiceCredentialChangeRequiresReview(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive, ReviewRequired: true}}
	supplierID := int64(7)
	proxyID := int64(11)
	accountRepo := &supplierAccountRepositoryStub{account: &Account{
		ID: 3, SupplierID: &supplierID, Name: "Account", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "old"}, Status: StatusActive, Schedulable: true,
		ReviewStatus: AccountReviewStatusApproved,
		ProxyID:      &proxyID,
	}}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), nil, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	credentials := map[string]any{"api_key": "new"}

	updated, err := svc.Update(context.Background(), 7, 3, UpdateSupplierAccountInput{Credentials: &credentials})
	require.NoError(t, err)
	require.Equal(t, "new", updated.Credentials["api_key"])
	require.Equal(t, AccountReviewStatusPending, updated.ReviewStatus)
	require.Equal(t, StatusDisabled, updated.Status)
	require.False(t, updated.Schedulable)
}

func TestSupplierCannotResumeAdministratorPausedAccount(t *testing.T) {
	proxyID, supplierID := int64(11), int64(7)
	supplier := &Supplier{ID: supplierID, Status: domain.SupplierStatusActive, ReviewRequired: true}
	repo := &supplierAccountRepositoryStub{account: &Account{ID: 3, SupplierID: &supplierID, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}, ProxyID: &proxyID, ReviewStatus: AccountReviewStatusApproved, Status: StatusDisabled}}
	svc := NewSupplierAccountService(repo, NewSupplierService(&supplierTokenRepositoryStub{supplier: supplier}), nil, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	active := StatusActive
	_, err := svc.Update(context.Background(), supplierID, 3, UpdateSupplierAccountInput{Status: &active})
	require.ErrorIs(t, err, ErrSupplierAccountInputInvalid)
	supplier.ReviewRequired = false
	updated, err := svc.Update(context.Background(), supplierID, 3, UpdateSupplierAccountInput{Status: &active})
	require.NoError(t, err)
	require.Equal(t, StatusActive, updated.Status)
}

func TestSupplierAccountServiceAdminEditPreservesSecretsAndReview(t *testing.T) {
	supplierID, proxyID := int64(7), int64(11)
	repo := &supplierAccountRepositoryStub{account: &Account{
		ID: 3, SupplierID: &supplierID, Name: "Original", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "secret", "base_url": "https://api.openai.com/"},
		Extra:       map[string]any{OllamaCloudUsageSessionExtraKey: map[string]any{"cursor": "preserve"}},
		Status:      StatusDisabled, Schedulable: false, ReviewStatus: AccountReviewStatusPending, ProxyID: &proxyID,
	}}
	svc := NewSupplierAccountService(repo, NewSupplierService(&supplierTokenRepositoryStub{supplier: &Supplier{ID: supplierID, Status: domain.SupplierStatusActive, ReviewRequired: true}}), nil,
		NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	name := "Reviewed configuration"
	rateMultiplier := 1.25
	credentials := map[string]any{"base_url": "https://api.openai.com/", "model_mapping": map[string]any{"gpt": "gpt-4"}}
	extra := map[string]any{"openai_compact_mode": OpenAICompactModeAuto}
	updated, err := svc.UpdateForAdmin(context.Background(), supplierID, 3, UpdateSupplierAccountInput{Name: &name, Credentials: &credentials, Extra: &extra, RateMultiplier: &rateMultiplier})
	require.NoError(t, err)
	require.Equal(t, "secret", updated.Credentials["api_key"])
	require.Equal(t, credentials["model_mapping"], updated.Credentials["model_mapping"])
	require.Equal(t, map[string]any{"cursor": "preserve"}, updated.Extra[OllamaCloudUsageSessionExtraKey])
	require.Equal(t, "Reviewed configuration", updated.Name)
	require.Equal(t, &rateMultiplier, updated.RateMultiplier)
	require.Equal(t, AccountReviewStatusPending, updated.ReviewStatus)
	require.Equal(t, StatusDisabled, updated.Status)
	require.False(t, updated.Schedulable)

	for _, key := range SensitiveCredentialKeys {
		credentials[key] = "replacement"
		_, err = svc.UpdateForAdmin(context.Background(), supplierID, 3, UpdateSupplierAccountInput{Credentials: &credentials})
		require.ErrorIs(t, err, ErrSupplierCredentialsInvalid)
		delete(credentials, key)
	}
	require.Equal(t, "secret", repo.account.Credentials["api_key"])

	active := StatusActive
	_, err = svc.UpdateForAdmin(context.Background(), supplierID, 3, UpdateSupplierAccountInput{Status: &active})
	require.ErrorIs(t, err, ErrSupplierAccountInputInvalid)
}

func TestUnapprovedSupplierAccountCannotBeScheduled(t *testing.T) {
	supplierID := int64(7)
	account := &Account{SupplierID: &supplierID, ReviewStatus: AccountReviewStatusPending, Status: StatusActive, Schedulable: true}
	require.False(t, account.IsSchedulable())
	account.ReviewStatus = AccountReviewStatusApproved
	require.True(t, account.IsSchedulable())
}

func TestSupplierAccountServiceRejectsOwnedAccountsAsOneBatch(t *testing.T) {
	repo := &supplierAccountRepositoryStub{accounts: []*Account{{ID: 3}, {ID: 4}}}
	svc := NewSupplierAccountService(repo, nil, nil, nil, &config.Config{}, nil)
	note := "invalid credential source"

	err := svc.Review(context.Background(), 7, SupplierAccountReviewInput{
		AccountIDs: []int64{3, 4, 4}, Action: SupplierAccountReviewReject, ReviewerID: 9, Note: &note,
	})
	require.NoError(t, err)
	require.Equal(t, []int64{3, 4}, repo.reviewed.AccountIDs)
	require.Equal(t, SupplierAccountReviewReject, repo.reviewed.Action)
}

func TestSupplierAccountServiceReviewRejectsMixedOwnership(t *testing.T) {
	repo := &supplierAccountRepositoryStub{accounts: []*Account{{ID: 3}}}
	svc := NewSupplierAccountService(repo, nil, nil, nil, &config.Config{}, nil)

	err := svc.Review(context.Background(), 7, SupplierAccountReviewInput{
		AccountIDs: []int64{3, 4}, Action: SupplierAccountReviewReject, ReviewerID: 9,
	})
	require.ErrorIs(t, err, ErrSupplierAccountNotFound)
	require.Nil(t, repo.reviewed)
}

func TestSupplierAccountServiceReviewRejectsOAuthOnlyGroupForAPIKey(t *testing.T) {
	proxyID := int64(11)
	repo := &supplierAccountRepositoryStub{accounts: []*Account{{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, ProxyID: &proxyID}}}
	svc := NewSupplierAccountService(repo, nil, &supplierGroupRepoStub{group: &Group{ID: 4, Platform: PlatformOpenAI, Status: StatusActive, RequireOAuthOnly: true}}, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	err := svc.Review(context.Background(), 7, SupplierAccountReviewInput{AccountIDs: []int64{3}, GroupIDs: []int64{4}, Action: SupplierAccountReviewApprove, ReviewerID: 9})
	require.ErrorIs(t, err, ErrSupplierAccountInputInvalid)
	require.Nil(t, repo.reviewed)
}

type supplierAccountRepositoryStub struct {
	created  *Account
	account  *Account
	accounts []*Account
	reviewed *SupplierAccountReviewInput
}

func (r *supplierAccountRepositoryStub) CreateOwned(_ context.Context, supplierID int64, account *Account, groups []AccountGroup) error {
	account.SupplierID = &supplierID
	account.ID = 1
	account.AccountGroups = groups
	r.created = account
	return nil
}

type supplierProxyRepoStub struct {
	ProxyRepository
	proxy *Proxy
}

func (r *supplierProxyRepoStub) GetByID(_ context.Context, id int64) (*Proxy, error) {
	if r.proxy == nil || r.proxy.ID != id {
		return nil, ErrProxyNotFound
	}
	return r.proxy, nil
}

func (r *supplierProxyRepoStub) ListActive(context.Context) ([]Proxy, error) {
	if r.proxy == nil {
		return nil, nil
	}
	return []Proxy{*r.proxy}, nil
}

type supplierGroupRepoStub struct {
	GroupRepository
	group  *Group
	groups map[int64]*Group
}

func (r *supplierGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if group := r.groups[id]; group != nil {
		return group, nil
	}
	if r.group == nil || r.group.ID != id {
		return nil, ErrGroupNotFound
	}
	return r.group, nil
}

func TestSupplierAccountServiceProxyFollowsInventory(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive, ReviewRequired: true, AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}}}
	accountRepo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), nil, NewProxyService(&supplierProxyRepoStub{}), &config.Config{}, nil)
	input := CreateSupplierAccountInput{Name: "test", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}}
	_, err := svc.Create(context.Background(), 7, input)
	require.NoError(t, err)
	require.Nil(t, accountRepo.created.ProxyID)
	proxyID := int64(99)
	input.ProxyID = &proxyID
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierProxyRequired)
	activeProxy := &supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}
	svc.proxies = NewProxyService(activeProxy)
	input.ProxyID = nil
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierProxyRequired)
	input.ProxyID = &proxyID
	_, err = svc.Create(context.Background(), 7, input)
	require.NoError(t, err)
}

func TestSupplierAccountWritePermissions(t *testing.T) {
	proxyID, groupID := int64(11), int64(42)
	supplier := &Supplier{ID: 7, Status: domain.SupplierStatusActive, ReviewRequired: true,
		AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}},
		AutoApproveGroups:   map[string][]int64{PlatformOpenAI: {groupID}}}
	repo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(repo, NewSupplierService(&supplierTokenRepositoryStub{supplier: supplier}),
		&supplierGroupRepoStub{group: &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive}},
		NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	input := CreateSupplierAccountInput{Name: "Account", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		ProxyID: &proxyID, Credentials: map[string]any{"api_key": "secret"}}
	concurrency := 10
	input.Concurrency = &concurrency
	_, err := svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAccountInputInvalid)
	input.Concurrency = nil
	input.Credentials["tier_id"] = "premium"
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierCredentialsInvalid)
	delete(input.Credentials, "tier_id")
	input.GroupIDs = []int64{groupID}
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAccountInputInvalid)
	input.GroupIDs = nil
	input.Extra = map[string]any{"openai_compact_mode": OpenAICompactModeForceOff}
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAccountInputInvalid)
	input.Extra = nil
	_, err = svc.Create(context.Background(), 7, input)
	require.NoError(t, err)

	supplier.ReviewRequired = false
	input.Concurrency = &concurrency
	input.GroupIDs = []int64{groupID + 1}
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAutoApproveGroupRequired)
	input.GroupIDs = []int64{groupID}
	input.Credentials["model_mapping"] = map[string]any{"public-model": "upstream-model"}
	input.Extra = map[string]any{"openai_compact_mode": OpenAICompactModeForceOff}
	account, err := svc.Create(context.Background(), 7, input)
	require.NoError(t, err)
	require.Equal(t, 10, account.Concurrency)
	require.Equal(t, []AccountGroup{{GroupID: groupID, Priority: 1}}, account.AccountGroups)
	require.Equal(t, 1.0, *account.RateMultiplier)
	require.Equal(t, map[string]any{"public-model": "upstream-model"}, account.Credentials["model_mapping"])
	require.Equal(t, OpenAICompactModeForceOff, account.Extra["openai_compact_mode"])
	repo.account = account
	_, err = svc.Update(context.Background(), 7, account.ID, UpdateSupplierAccountInput{GroupIDs: []int64{groupID + 1}})
	require.ErrorIs(t, err, ErrSupplierAutoApproveGroupRequired)
	updated, err := svc.Update(context.Background(), 7, account.ID, UpdateSupplierAccountInput{GroupIDs: []int64{groupID}})
	require.NoError(t, err)
	require.Equal(t, []int64{groupID}, updated.GroupIDs)
	repo.account.Extra["managed_state"] = "preserved"
	extra := map[string]any{"openai_compact_mode": OpenAICompactModeAuto}
	updated, err = svc.Update(context.Background(), 7, account.ID, UpdateSupplierAccountInput{Extra: &extra})
	require.NoError(t, err)
	require.Equal(t, "preserved", updated.Extra["managed_state"])
	require.Equal(t, OpenAICompactModeAuto, updated.Extra["openai_compact_mode"])
	mappingOnly := map[string]any{"model_mapping": map[string]any{"public-model": "new-upstream-model"}}
	updated, err = svc.Update(context.Background(), 7, account.ID, UpdateSupplierAccountInput{Credentials: &mappingOnly})
	require.NoError(t, err)
	require.Equal(t, "secret", updated.Credentials["api_key"])
	require.Equal(t, mappingOnly["model_mapping"], updated.Credentials["model_mapping"])
	require.Equal(t, AccountReviewStatusApproved, updated.ReviewStatus)
	input.Extra = map[string]any{"openai_passthrough": true}
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAccountInputInvalid)
	input.Extra = nil
	input.Credentials["model_mapping"] = map[string]any{"public-model": 123}
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierCredentialsInvalid)
}

func TestTrustedSupplierRequiresConfiguredPlatformGroup(t *testing.T) {
	proxyID, groupID := int64(11), int64(42)
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive, AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}}}
	accountRepo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), &supplierGroupRepoStub{group: &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive}}, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	input := CreateSupplierAccountInput{Name: "test", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}, ProxyID: &proxyID}
	_, err := svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAutoApproveGroupRequired)
	supplierRepo.supplier.AutoApproveGroups = map[string][]int64{PlatformOpenAI: {groupID}}
	account, err := svc.Create(context.Background(), 7, input)
	require.NoError(t, err)
	require.Equal(t, AccountReviewStatusApproved, account.ReviewStatus)
	require.Equal(t, StatusActive, account.Status)
	require.True(t, account.Schedulable)
	require.Equal(t, []AccountGroup{{GroupID: groupID, Priority: 1}}, account.AccountGroups)
}

func TestSupplierCanSaveNoReviewPolicyWithoutGroupsButCannotAutoApprove(t *testing.T) {
	supplier := &Supplier{ID: 7, Status: domain.SupplierStatusActive, ReviewRequired: true}
	repo := &supplierTokenRepositoryStub{supplier: supplier}
	svc := NewSupplierService(repo)
	kinds := []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}
	reviewRequired := false
	groups := map[string][]int64{}
	updated, err := svc.Update(context.Background(), supplier.ID, UpdateSupplierInput{
		AllowedAccountKinds: &kinds, ReviewRequired: &reviewRequired, AutoApproveGroups: &groups,
	})
	require.NoError(t, err)
	require.False(t, updated.ReviewRequired)
	require.Equal(t, kinds, updated.AllowedAccountKinds)
	require.Empty(t, updated.AutoApproveGroups)
	proxyID := int64(11)
	accountRepo := &supplierAccountRepositoryStub{}
	accountService := NewSupplierAccountService(accountRepo, svc, nil,
		NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	_, err = accountService.Create(context.Background(), supplier.ID, CreateSupplierAccountInput{
		Name: "unroutable", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "secret"}, ProxyID: &proxyID,
	})
	require.ErrorIs(t, err, ErrSupplierAutoApproveGroupRequired)
	require.Nil(t, accountRepo.created)
}

func TestTrustedSupplierCanSelectSeveralAuthorizedGroups(t *testing.T) {
	proxyID := int64(11)
	supplier := &Supplier{ID: 7, Status: domain.SupplierStatusActive, AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}, AutoApproveGroups: map[string][]int64{PlatformOpenAI: {42, 43}}}
	groups := &supplierGroupRepoStub{groups: map[int64]*Group{42: {ID: 42, Platform: PlatformOpenAI, Status: StatusActive}, 43: {ID: 43, Platform: PlatformOpenAI, Status: StatusActive}}}
	repo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(repo, NewSupplierService(&supplierTokenRepositoryStub{supplier: supplier}), groups, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	input := CreateSupplierAccountInput{Name: "test", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}, ProxyID: &proxyID, GroupIDs: []int64{42, 43}}
	account, err := svc.Create(context.Background(), 7, input)
	require.NoError(t, err)
	require.Equal(t, []AccountGroup{{GroupID: 42, Priority: 1}, {GroupID: 43, Priority: 1}}, account.AccountGroups)
	repo.account = account
	updated, err := svc.Update(context.Background(), 7, account.ID, UpdateSupplierAccountInput{GroupIDs: []int64{43, 42}})
	require.NoError(t, err)
	require.Equal(t, []int64{43, 42}, updated.GroupIDs)
	require.Equal(t, []AccountGroup{{GroupID: 43, Priority: 1}, {GroupID: 42, Priority: 1}}, updated.AccountGroups)
	input.GroupIDs = []int64{42, 42}
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAutoApproveGroupRequired)
	input.GroupIDs = []int64{42, 44}
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAutoApproveGroupRequired)
	groups.groups[42].RequireOAuthOnly = true
	input.GroupIDs = nil
	account, err = svc.Create(context.Background(), 7, input)
	require.NoError(t, err)
	require.Equal(t, []AccountGroup{{GroupID: 43, Priority: 1}}, account.AccountGroups)
}

func TestSupplierAuthorizedGroupsOnlyExposesActiveAuthorizedNames(t *testing.T) {
	supplier := &Supplier{AutoApproveGroups: map[string][]int64{PlatformOpenAI: {42, 43, 44, 45}}}
	groups := &supplierGroupRepoStub{groups: map[int64]*Group{
		42: {ID: 42, Name: "Standard", Description: "Normal traffic", Platform: PlatformOpenAI, Status: StatusActive},
		43: {ID: 43, Name: "Other platform", Platform: PlatformAnthropic, Status: StatusActive},
		44: {ID: 44, Name: "Disabled", Platform: PlatformOpenAI, Status: StatusDisabled},
		45: {ID: 45, Name: "OAuth pool", Platform: PlatformOpenAI, Status: StatusActive, RequireOAuthOnly: true},
	}}
	service := &SupplierAccountService{groups: groups}
	options, err := service.AuthorizedGroups(context.Background(), supplier)
	require.NoError(t, err)
	require.Equal(t, []SupplierGroupOption{
		{ID: 42, Name: "Standard", Description: "Normal traffic", Platform: PlatformOpenAI},
		{ID: 45, Name: "OAuth pool", Platform: PlatformOpenAI, RequireOAuthOnly: true},
	}, options)
	supplier.ReviewRequired = true
	options, err = service.AuthorizedGroups(context.Background(), supplier)
	require.NoError(t, err)
	require.Empty(t, options)
}
func (r *supplierAccountRepositoryStub) GetOwnedByID(context.Context, int64, int64) (*Account, error) {
	if r.account == nil {
		return nil, ErrSupplierAccountNotFound
	}
	clone := *r.account
	return &clone, nil
}
func (r *supplierAccountRepositoryStub) GetOwnedByIDs(context.Context, int64, []int64) ([]*Account, error) {
	return r.accounts, nil
}
func (r *supplierAccountRepositoryStub) ListOwned(context.Context, int64, pagination.PaginationParams, SupplierAccountFilters) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *supplierAccountRepositoryStub) UpdateOwned(_ context.Context, _ int64, account *Account, groups []AccountGroup) error {
	clone := *account
	if groups != nil {
		clone.AccountGroups = groups
	}
	r.account = &clone
	return nil
}
func (r *supplierAccountRepositoryStub) DeleteOwned(context.Context, int64, int64) error { return nil }

func (r *supplierAccountRepositoryStub) ReviewOwned(_ context.Context, _ int64, input SupplierAccountReviewInput) error {
	r.reviewed = &input
	return nil
}
