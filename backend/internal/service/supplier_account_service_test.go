package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestSupplierAccountServiceCreateUsesSafePendingDefaults(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{
		ID: 7, Status: domain.SupplierStatusActive,
		AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}},
	}}
	accountRepo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), nil, &config.Config{})

	account, err := svc.Create(context.Background(), 7, CreateSupplierAccountInput{
		ExternalID: " vendor-1 ", Name: " Primary ", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": " secret ", "base_url": "https://api.openai.com/"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(7), *account.SupplierID)
	require.Equal(t, "vendor-1", *account.SupplierExternalID)
	require.Equal(t, AccountReviewStatusPending, account.ReviewStatus)
	require.Equal(t, StatusDisabled, account.Status)
	require.False(t, account.Schedulable)
	require.Equal(t, "secret", account.Credentials["api_key"])
}

func TestSupplierAccountServiceCreateRejectsKindNotAllowedForSupplier(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive}}
	svc := NewSupplierAccountService(&supplierAccountRepositoryStub{}, NewSupplierService(supplierRepo), nil, &config.Config{})

	_, err := svc.Create(context.Background(), 7, CreateSupplierAccountInput{
		Name: "Primary", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "secret"},
	})
	require.ErrorIs(t, err, ErrSupplierAccountKindNotAllowed)
}

func TestSupplierAccountServiceCredentialChangeRequiresReview(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{ID: 7, Status: domain.SupplierStatusActive}}
	supplierID := int64(7)
	accountRepo := &supplierAccountRepositoryStub{account: &Account{
		ID: 3, SupplierID: &supplierID, Name: "Account", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "old"}, Status: StatusActive, Schedulable: true,
		ReviewStatus: AccountReviewStatusApproved,
	}}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), nil, &config.Config{})
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
	svc := NewSupplierAccountService(repo, nil, nil, &config.Config{})
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
	svc := NewSupplierAccountService(repo, nil, nil, &config.Config{})

	err := svc.Review(context.Background(), 7, SupplierAccountReviewInput{
		AccountIDs: []int64{3, 4}, Action: SupplierAccountReviewReject, ReviewerID: 9,
	})
	require.ErrorIs(t, err, ErrSupplierAccountNotFound)
	require.Nil(t, repo.reviewed)
}

type supplierAccountRepositoryStub struct {
	created  *Account
	account  *Account
	accounts []*Account
	reviewed *SupplierAccountReviewInput
}

func (r *supplierAccountRepositoryStub) CreateOwned(_ context.Context, supplierID int64, account *Account) error {
	account.SupplierID = &supplierID
	account.ID = 1
	r.created = account
	return nil
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
