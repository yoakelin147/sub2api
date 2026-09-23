package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type supplierPasswordPersistenceRepo struct {
	AccountRepository
	credentials map[string]any
}

func (r *supplierPasswordPersistenceRepo) UpdateCredentials(_ context.Context, _ int64, credentials map[string]any) error {
	r.credentials = credentials
	return nil
}

func TestSupplierLoginPasswordSurvivesTokenRefresh(t *testing.T) {
	supplierID := int64(7)
	account := &Account{ID: 3, SupplierID: &supplierID, Credentials: map[string]any{"access_token": "old", "login_password_encrypted": "ciphertext"}}
	repo := &supplierPasswordPersistenceRepo{}
	updated := map[string]any{"access_token": "new"}
	require.NoError(t, persistAccountCredentials(context.Background(), repo, account, updated))
	require.Equal(t, map[string]any{"access_token": "new", "login_password_encrypted": "ciphertext"}, repo.credentials)
	require.Equal(t, map[string]any{"access_token": "new"}, updated)
}
