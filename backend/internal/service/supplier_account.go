package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var ErrSupplierAccountNotFound = infraerrors.NotFound(
	"SUPPLIER_ACCOUNT_NOT_FOUND",
	"supplier account not found",
)

type SupplierAccountFilters struct {
	Platform     string
	Type         string
	Status       string
	ReviewStatus string
	Search       string
}

// SupplierAccountRepository exposes only tenant-scoped account persistence.
type SupplierAccountRepository interface {
	CreateOwned(ctx context.Context, supplierID int64, account *Account) error
	GetOwnedByID(ctx context.Context, supplierID, accountID int64) (*Account, error)
	GetOwnedByIDs(ctx context.Context, supplierID int64, accountIDs []int64) ([]*Account, error)
	ListOwned(ctx context.Context, supplierID int64, params pagination.PaginationParams, filters SupplierAccountFilters) ([]Account, *pagination.PaginationResult, error)
	UpdateOwned(ctx context.Context, supplierID int64, account *Account) error
	DeleteOwned(ctx context.Context, supplierID, accountID int64) error
}
