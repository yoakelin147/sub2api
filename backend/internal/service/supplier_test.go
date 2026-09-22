package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSupplierAccountKinds(t *testing.T) {
	got, err := NormalizeSupplierAccountKinds([]domain.SupplierAccountKind{
		{Platform: " OpenAI ", Type: " APIKEY "},
		{Platform: "openai", Type: "apikey"},
		{Platform: "anthropic", Type: "bedrock"},
		{Platform: "gemini", Type: "service_account"},
	})
	require.NoError(t, err)
	require.Equal(t, []domain.SupplierAccountKind{
		{Platform: "openai", Type: "apikey"},
		{Platform: "anthropic", Type: "bedrock"},
		{Platform: "gemini", Type: "service_account"},
	}, got)
}

func TestSupplierServiceCreateNormalizesInput(t *testing.T) {
	repo := &supplierRepositoryStub{}
	svc := NewSupplierService(repo)

	created, err := svc.Create(context.Background(), CreateSupplierInput{
		Code: " Demo_One ",
		Name: " Demo Supplier ",
		AllowedAccountKinds: []SupplierAccountKind{
			{Platform: " OpenAI ", Type: " APIKEY "},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "demo_one", created.Code)
	require.Equal(t, "Demo Supplier", created.Name)
	require.Equal(t, domain.SupplierStatusActive, created.Status)
	require.Equal(t, []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}, created.AllowedAccountKinds)
}

func TestSupplierServiceCreateRejectsInvalidCodeAndStatus(t *testing.T) {
	svc := NewSupplierService(&supplierRepositoryStub{})

	_, err := svc.Create(context.Background(), CreateSupplierInput{Code: "not valid!", Name: "Supplier"})
	require.ErrorIs(t, err, ErrSupplierCodeInvalid)

	_, err = svc.Create(context.Background(), CreateSupplierInput{
		Code: "valid", Name: "Supplier", Status: "deleted",
	})
	require.ErrorIs(t, err, ErrSupplierStatusInvalid)
}

type supplierRepositoryStub struct {
	created *Supplier
}

func (r *supplierRepositoryStub) Create(_ context.Context, supplier *Supplier) error {
	supplier.ID = 1
	r.created = supplier
	return nil
}

func (r *supplierRepositoryStub) GetByID(context.Context, int64) (*Supplier, error) {
	return nil, ErrSupplierNotFound
}

func (r *supplierRepositoryStub) GetByTokenSelector(context.Context, string) (*Supplier, error) {
	return nil, ErrSupplierNotFound
}

func (r *supplierRepositoryStub) Update(context.Context, *Supplier) error { return nil }
func (r *supplierRepositoryStub) Delete(context.Context, int64) error     { return nil }

func (r *supplierRepositoryStub) List(
	context.Context,
	pagination.PaginationParams,
	SupplierListFilters,
) ([]Supplier, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *supplierRepositoryStub) UpdateTokenLastUsed(context.Context, int64, time.Time) error {
	return nil
}

func TestNormalizeSupplierAccountKindsRejectsInternalAndUnknownKinds(t *testing.T) {
	tests := []domain.SupplierAccountKind{
		{Platform: PlatformComposite, Type: AccountTypeAPIKey},
		{Platform: PlatformOpenAI, Type: AccountTypeBedrock},
		{Platform: "unknown", Type: AccountTypeAPIKey},
		{Platform: PlatformOpenAI, Type: "cookie"},
	}

	for _, kind := range tests {
		_, err := NormalizeSupplierAccountKinds([]domain.SupplierAccountKind{kind})
		require.ErrorIs(t, err, ErrSupplierAccountKindInvalid)
	}
}
