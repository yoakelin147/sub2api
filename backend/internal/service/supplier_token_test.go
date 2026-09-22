package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestSupplierTokenGenerateAuthenticateRotateAndRevoke(t *testing.T) {
	repo := &supplierTokenRepositoryStub{supplier: &Supplier{
		ID: 7, Code: "demo", Name: "Demo", Status: domain.SupplierStatusActive,
	}}
	svc := NewSupplierTokenService(repo)

	first, err := svc.Regenerate(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(first, "supplier_"))
	require.NotContains(t, *repo.supplier.TokenHash, first)
	require.NotContains(t, *repo.supplier.TokenHash, strings.TrimPrefix(first, "supplier_"))

	authenticated, err := svc.Authenticate(context.Background(), first)
	require.NoError(t, err)
	require.Equal(t, int64(7), authenticated.ID)

	second, err := svc.Regenerate(context.Background(), 7)
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	_, err = svc.Authenticate(context.Background(), first)
	require.ErrorIs(t, err, ErrSupplierTokenInvalid)

	require.NoError(t, svc.Revoke(context.Background(), 7))
	_, err = svc.Authenticate(context.Background(), second)
	require.ErrorIs(t, err, ErrSupplierTokenInvalid)
}

func TestSupplierTokenRejectsDisabledSupplier(t *testing.T) {
	repo := &supplierTokenRepositoryStub{supplier: &Supplier{
		ID: 8, Code: "disabled", Name: "Disabled", Status: domain.SupplierStatusActive,
	}}
	svc := NewSupplierTokenService(repo)
	token, err := svc.Regenerate(context.Background(), 8)
	require.NoError(t, err)
	repo.supplier.Status = domain.SupplierStatusDisabled

	_, err = svc.Authenticate(context.Background(), token)
	require.ErrorIs(t, err, ErrSupplierDisabled)
}

type supplierTokenRepositoryStub struct {
	supplier *Supplier
}

func (r *supplierTokenRepositoryStub) Create(context.Context, *Supplier) error { return nil }
func (r *supplierTokenRepositoryStub) GetByID(_ context.Context, id int64) (*Supplier, error) {
	if r.supplier == nil || r.supplier.ID != id {
		return nil, ErrSupplierNotFound
	}
	clone := *r.supplier
	return &clone, nil
}
func (r *supplierTokenRepositoryStub) GetByTokenSelector(_ context.Context, selector string) (*Supplier, error) {
	if r.supplier == nil || r.supplier.TokenSelector == nil || *r.supplier.TokenSelector != selector {
		return nil, ErrSupplierNotFound
	}
	clone := *r.supplier
	return &clone, nil
}
func (r *supplierTokenRepositoryStub) Update(_ context.Context, supplier *Supplier) error {
	clone := *supplier
	r.supplier = &clone
	return nil
}
func (r *supplierTokenRepositoryStub) Delete(context.Context, int64) error { return nil }
func (r *supplierTokenRepositoryStub) List(context.Context, pagination.PaginationParams, SupplierListFilters) ([]Supplier, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *supplierTokenRepositoryStub) GetStatsBySupplierIDs(context.Context, []int64) (map[int64]SupplierStats, error) {
	return map[int64]SupplierStats{}, nil
}
func (r *supplierTokenRepositoryStub) UpdateTokenLastUsed(_ context.Context, id int64, usedAt time.Time) error {
	if r.supplier != nil && r.supplier.ID == id {
		r.supplier.TokenLastUsedAt = &usedAt
	}
	return nil
}
