package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestSupplierAccountRepositoryGetOwnedByIDScopesQueryToSupplier(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "accounts".`)).
		WithArgs(int64(27), int64(9)).
		WillReturnRows(updatedAccountRows(27, `{}`))

	repo := newAccountRepositoryWithSQL(client, db, nil)
	account, err := repo.GetOwnedByID(context.Background(), 9, 27)
	require.NoError(t, err)
	require.Equal(t, int64(27), account.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

var _ sqlExecutor = (*sql.DB)(nil)
