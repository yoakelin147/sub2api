package repository

import (
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierEntityToServicePreservesTenantConfiguration(t *testing.T) {
	now := time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)
	prefix := "supplier_demo"
	row := &dbent.Supplier{
		ID:             7,
		Code:           "demo",
		Name:           "Demo Supplier",
		Status:         domain.SupplierStatusActive,
		ReviewRequired: true,
		AllowedAccountKinds: []domain.SupplierAccountKind{
			{Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey},
		},
		TokenPrefix: &prefix,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	got := supplierEntityToService(row)
	require.Equal(t, &service.Supplier{
		ID:             7,
		Code:           "demo",
		Name:           "Demo Supplier",
		Status:         domain.SupplierStatusActive,
		ReviewRequired: true,
		AllowedAccountKinds: []domain.SupplierAccountKind{
			{Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey},
		},
		TokenPrefix: &prefix,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, got)
}

func TestAccountAndUserEntityMappersPreserveSupplierOwnership(t *testing.T) {
	supplierID := int64(9)
	reviewerID := int64(3)
	externalID := "vendor-account-7"
	reviewNote := "verified"

	account := accountEntityToService(&dbent.Account{
		ID:                 7,
		SupplierID:         &supplierID,
		SupplierExternalID: &externalID,
		ReviewStatus:       domain.AccountReviewStatusApproved,
		ReviewedBy:         &reviewerID,
		ReviewNote:         &reviewNote,
	})
	require.Equal(t, &supplierID, account.SupplierID)
	require.Equal(t, &externalID, account.SupplierExternalID)
	require.Equal(t, domain.AccountReviewStatusApproved, account.ReviewStatus)
	require.Equal(t, &reviewerID, account.ReviewedBy)
	require.Equal(t, &reviewNote, account.ReviewNote)

	user := userEntityToService(&dbent.User{ID: 11, SupplierID: &supplierID})
	require.Equal(t, &supplierID, user.SupplierID)
}
