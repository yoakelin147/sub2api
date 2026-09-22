package repository

import (
	"context"
	"errors"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func NewSupplierAccountRepository(
	client *dbent.Client,
	sqlq sqlExecutor,
	schedulerCache service.SchedulerCache,
) service.SupplierAccountRepository {
	return newAccountRepositoryWithSQL(client, sqlq, schedulerCache)
}

func (r *accountRepository) CreateOwned(ctx context.Context, supplierID int64, account *service.Account) error {
	account.SupplierID = &supplierID
	return r.CreateWithAccountGroups(ctx, account, nil)
}

func (r *accountRepository) GetOwnedByID(ctx context.Context, supplierID, accountID int64) (*service.Account, error) {
	row, err := r.client.Account.Query().
		Where(dbaccount.IDEQ(accountID), dbaccount.SupplierIDEQ(supplierID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupplierAccountNotFound, nil)
	}
	return accountEntityToService(row), nil
}

func (r *accountRepository) GetOwnedByIDs(ctx context.Context, supplierID int64, accountIDs []int64) ([]*service.Account, error) {
	if len(accountIDs) == 0 {
		return []*service.Account{}, nil
	}
	rows, err := r.client.Account.Query().
		Where(dbaccount.IDIn(accountIDs...), dbaccount.SupplierIDEQ(supplierID)).
		Order(dbent.Asc(dbaccount.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*service.Account, 0, len(rows))
	for _, row := range rows {
		result = append(result, accountEntityToService(row))
	}
	return result, nil
}

func (r *accountRepository) ListOwned(
	ctx context.Context,
	supplierID int64,
	params pagination.PaginationParams,
	filters service.SupplierAccountFilters,
) ([]service.Account, *pagination.PaginationResult, error) {
	query := r.client.Account.Query().Where(dbaccount.SupplierIDEQ(supplierID))
	if filters.Platform != "" {
		query = query.Where(dbaccount.PlatformEQ(filters.Platform))
	}
	if filters.Type != "" {
		query = query.Where(dbaccount.TypeEQ(filters.Type))
	}
	if filters.Status != "" {
		query = query.Where(dbaccount.StatusEQ(filters.Status))
	}
	if filters.ReviewStatus != "" {
		query = query.Where(dbaccount.ReviewStatusEQ(filters.ReviewStatus))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		query = query.Where(dbaccount.Or(
			dbaccount.NameContainsFold(search),
			dbaccount.SupplierExternalIDContainsFold(search),
		))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := query.Order(dbent.Desc(dbaccount.FieldCreatedAt), dbent.Desc(dbaccount.FieldID)).
		Offset(params.Offset()).Limit(params.Limit()).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	items := make([]service.Account, 0, len(rows))
	for _, row := range rows {
		items = append(items, *accountEntityToService(row))
	}
	return items, paginationResultFromTotal(int64(total), params), nil
}

func (r *accountRepository) UpdateOwned(ctx context.Context, supplierID int64, account *service.Account) error {
	baseCtx := ctx
	client := r.client
	var tx *dbent.Tx
	if contextTx := dbent.TxFromContext(ctx); contextTx != nil {
		client = contextTx.Client()
	} else {
		var err error
		tx, err = r.client.Tx(ctx)
		if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
			return err
		}
		if tx != nil {
			defer func() { _ = tx.Rollback() }()
			ctx = dbent.NewTxContext(ctx, tx)
			client = tx.Client()
		}
	}
	update := client.Account.Update().
		Where(dbaccount.IDEQ(account.ID), dbaccount.SupplierIDEQ(supplierID)).
		SetName(account.Name).
		SetNillableNotes(account.Notes).
		SetCredentials(normalizeJSONMap(account.Credentials)).
		SetStatus(account.Status).
		SetSchedulable(account.Schedulable).
		SetReviewStatus(account.ReviewStatus).
		SetNillableReviewedAt(account.ReviewedAt).
		SetNillableReviewedBy(account.ReviewedBy).
		SetNillableReviewNote(account.ReviewNote).
		SetNillableExpiresAt(account.ExpiresAt).
		SetNillableSupplierExternalID(account.SupplierExternalID)
	if account.Notes == nil {
		update.ClearNotes()
	}
	if account.SupplierExternalID == nil {
		update.ClearSupplierExternalID()
	}
	if account.ExpiresAt == nil {
		update.ClearExpiresAt()
	}
	if account.ReviewedAt == nil {
		update.ClearReviewedAt()
	}
	if account.ReviewedBy == nil {
		update.ClearReviewedBy()
	}
	if account.ReviewNote == nil {
		update.ClearReviewNote()
	}
	count, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		return service.ErrSupplierAccountNotFound
	}
	if err := enqueueSchedulerOutbox(ctx, client, service.SchedulerOutboxEventAccountChanged, &account.ID, nil, nil); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if dbent.TxFromContext(baseCtx) == nil {
		r.syncSchedulerAccountSnapshot(baseCtx, account.ID)
	}
	return nil
}

func (r *accountRepository) DeleteOwned(ctx context.Context, supplierID, accountID int64) error {
	baseCtx := ctx
	client := r.client
	var tx *dbent.Tx
	if contextTx := dbent.TxFromContext(ctx); contextTx != nil {
		client = contextTx.Client()
	} else {
		var err error
		tx, err = r.client.Tx(ctx)
		if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
			return err
		}
		if tx != nil {
			defer func() { _ = tx.Rollback() }()
			ctx = dbent.NewTxContext(ctx, tx)
			client = tx.Client()
		}
	}
	count, err := client.Account.Delete().
		Where(dbaccount.IDEQ(accountID), dbaccount.SupplierIDEQ(supplierID)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		return service.ErrSupplierAccountNotFound
	}
	if err := enqueueSchedulerOutbox(ctx, client, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
		return err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	if dbent.TxFromContext(baseCtx) == nil {
		r.syncSchedulerAccountSnapshot(baseCtx, accountID)
	}
	return nil
}
