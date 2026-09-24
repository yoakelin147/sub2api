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
	mock.ExpectQuery(regexp.QuoteMeta(`FROM "account_groups" WHERE "account_groups"."account_id" IN ($1)`)).
		WithArgs(int64(27)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "group_id", "priority", "created_at"}))

	repo := newAccountRepositoryWithSQL(client, db, nil)
	account, err := repo.GetOwnedByID(context.Background(), 9, 27)
	require.NoError(t, err)
	require.Equal(t, int64(27), account.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

var _ sqlExecutor = (*sql.DB)(nil)

func TestSupplierAccountEntityIncludesEveryAssignedGroup(t *testing.T) {
	account := supplierAccountEntityToService(&dbent.Account{
		ID: 27,
		Edges: dbent.AccountEdges{AccountGroups: []*dbent.AccountGroup{
			{AccountID: 27, GroupID: 3, Priority: 1},
			{AccountID: 27, GroupID: 4, Priority: 1},
		}},
	})
	require.Equal(t, []int64{3, 4}, account.GroupIDs)
}
