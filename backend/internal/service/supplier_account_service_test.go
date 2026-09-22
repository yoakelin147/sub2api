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
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), &config.Config{})

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
	svc := NewSupplierAccountService(&supplierAccountRepositoryStub{}, NewSupplierService(supplierRepo), &config.Config{})

	_, err := svc.Create(context.Background(), 7, CreateSupplierAccountInput{
		Name: "Primary", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "secret"},
	})
	require.ErrorIs(t, err, ErrSupplierAccountKindNotAllowed)
}

type supplierAccountRepositoryStub struct {
	created *Account
}

func (r *supplierAccountRepositoryStub) CreateOwned(_ context.Context, supplierID int64, account *Account) error {
	account.SupplierID = &supplierID
	account.ID = 1
	r.created = account
	return nil
}
func (r *supplierAccountRepositoryStub) GetOwnedByID(context.Context, int64, int64) (*Account, error) {
	return nil, ErrSupplierAccountNotFound
}
func (r *supplierAccountRepositoryStub) GetOwnedByIDs(context.Context, int64, []int64) ([]*Account, error) {
	return nil, nil
}
func (r *supplierAccountRepositoryStub) ListOwned(context.Context, int64, pagination.PaginationParams, SupplierAccountFilters) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *supplierAccountRepositoryStub) UpdateOwned(context.Context, int64, *Account) error {
	return nil
}
func (r *supplierAccountRepositoryStub) DeleteOwned(context.Context, int64, int64) error { return nil }
