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
	group *Group
}

func (r *supplierGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	if r.group == nil || r.group.ID != id {
		return nil, ErrGroupNotFound
	}
	return r.group, nil
}

func TestSupplierAccountServiceRequiresInventoryProxy(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive, ReviewRequired: true, AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}}}
	accountRepo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), nil, NewProxyService(&supplierProxyRepoStub{}), &config.Config{}, nil)
	input := CreateSupplierAccountInput{Name: "test", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}}
	_, err := svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierProxyRequired)
	proxyID := int64(99)
	input.ProxyID = &proxyID
	_, err = svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierProxyRequired)
	require.Nil(t, accountRepo.created)
}

func TestTrustedSupplierRequiresConfiguredPlatformGroup(t *testing.T) {
	proxyID, groupID := int64(11), int64(42)
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive, AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}}}
	accountRepo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), &supplierGroupRepoStub{group: &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive}}, NewProxyService(&supplierProxyRepoStub{proxy: &Proxy{ID: proxyID, Status: StatusActive}}), &config.Config{}, nil)
	input := CreateSupplierAccountInput{Name: "test", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}, ProxyID: &proxyID}
	_, err := svc.Create(context.Background(), 7, input)
	require.ErrorIs(t, err, ErrSupplierAutoApproveGroupRequired)
	supplierRepo.supplier.AutoApproveGroups = map[string]int64{PlatformOpenAI: groupID}
	account, err := svc.Create(context.Background(), 7, input)
	require.NoError(t, err)
	require.Equal(t, AccountReviewStatusApproved, account.ReviewStatus)
	require.Equal(t, StatusActive, account.Status)
	require.True(t, account.Schedulable)
	require.Equal(t, []AccountGroup{{GroupID: groupID, Priority: 1}}, account.AccountGroups)
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

func (r *supplierAccountRepositoryStub) UpdateOwned(_ context.Context, _ int64, account *Account) error {
	clone := *account
	r.account = &clone
	return nil
}
func (r *supplierAccountRepositoryStub) DeleteOwned(context.Context, int64, int64) error { return nil }

func (r *supplierAccountRepositoryStub) ReviewOwned(_ context.Context, _ int64, input SupplierAccountReviewInput) error {
	r.reviewed = &input
	return nil
}
