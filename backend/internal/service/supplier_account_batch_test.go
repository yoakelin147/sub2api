package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSupplierAccountServiceBatchCreateReturnsPerItemResults(t *testing.T) {
	supplierRepo := &supplierTokenRepositoryStub{supplier: &Supplier{
		ID: 7, Status: domain.SupplierStatusActive,
		AllowedAccountKinds: []SupplierAccountKind{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}},
	}}
	accountRepo := &supplierAccountRepositoryStub{}
	svc := NewSupplierAccountService(accountRepo, NewSupplierService(supplierRepo), &config.Config{})

	result, err := svc.BatchCreate(context.Background(), 7, []CreateSupplierAccountInput{
		{ExternalID: "good", Name: "Good", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}},
		{ExternalID: "bad", Name: "Bad", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 1, result.Failed)
	require.True(t, result.Results[0].Success)
	require.False(t, result.Results[1].Success)
}

func TestSupplierAccountServiceBatchCreateRejectsMoreThanLimit(t *testing.T) {
	svc := NewSupplierAccountService(&supplierAccountRepositoryStub{}, NewSupplierService(&supplierTokenRepositoryStub{}), &config.Config{})
	items := make([]CreateSupplierAccountInput, MaxSupplierAccountBatchSize+1)
	for i := range items {
		items[i].ExternalID = fmt.Sprintf("item-%d", i)
	}

	_, err := svc.BatchCreate(context.Background(), 7, items)
	require.ErrorIs(t, err, ErrSupplierAccountBatchTooLarge)
}
