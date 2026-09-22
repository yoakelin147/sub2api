package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func NewSupplierAccountRepository(
	client *dbent.Client,
	sqlDB *sql.DB,
	schedulerCache service.SchedulerCache,
) service.SupplierAccountRepository {
	return newAccountRepositoryWithSQL(client, sqlDB, schedulerCache)
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

func (r *accountRepository) ReviewOwned(ctx context.Context, supplierID int64, input service.SupplierAccountReviewInput) error {
	baseCtx := ctx
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	ctx = dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	rows, err := client.Account.Query().
		Where(dbaccount.IDIn(input.AccountIDs...), dbaccount.SupplierIDEQ(supplierID)).
		Select(dbaccount.FieldID).
		All(ctx)
	if err != nil {
		return err
	}
	if len(rows) != len(input.AccountIDs) {
		return service.ErrSupplierAccountNotFound
	}

	update := client.Account.Update().Where(
		dbaccount.IDIn(input.AccountIDs...),
		dbaccount.SupplierIDEQ(supplierID),
	)
	now := time.Now().UTC()
	switch input.Action {
	case service.SupplierAccountReviewApprove:
		if err := lockLiveGroups(ctx, client, input.GroupIDs); err != nil {
			return err
		}
		if _, err := client.AccountGroup.Delete().Where(dbaccountgroup.AccountIDIn(input.AccountIDs...)).Exec(ctx); err != nil {
			return err
		}
		builders := make([]*dbent.AccountGroupCreate, 0, len(input.AccountIDs)*len(input.GroupIDs))
		for _, accountID := range input.AccountIDs {
			for priority, groupID := range input.GroupIDs {
				builders = append(builders, client.AccountGroup.Create().SetAccountID(accountID).SetGroupID(groupID).SetPriority(priority+1))
			}
		}
		if len(builders) > 0 {
			if _, err := client.AccountGroup.CreateBulk(builders...).Save(ctx); err != nil {
				return err
			}
		}
		update.SetReviewStatus(service.AccountReviewStatusApproved).
			SetStatus(service.StatusActive).
			SetSchedulable(true).
			SetReviewedAt(now).
			SetReviewedBy(input.ReviewerID).
			SetNillableReviewNote(input.Note)
		if input.Note == nil {
			update.ClearReviewNote()
		}
	case service.SupplierAccountReviewReject:
		update.SetReviewStatus(service.AccountReviewStatusRejected).
			SetStatus(service.StatusDisabled).
			SetSchedulable(false).
			SetReviewedAt(now).
			SetReviewedBy(input.ReviewerID).
			SetNillableReviewNote(input.Note)
		if input.Note == nil {
			update.ClearReviewNote()
		}
	case service.SupplierAccountReviewPause:
		update.SetStatus(service.StatusDisabled).SetSchedulable(false)
	default:
		return service.ErrSupplierAccountInputInvalid
	}
	count, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if count != len(input.AccountIDs) {
		return service.ErrSupplierAccountNotFound
	}
	payload := map[string]any{"account_ids": input.AccountIDs}
	if err := enqueueSchedulerOutbox(ctx, client, service.SchedulerOutboxEventAccountBulkChanged, nil, nil, payload); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.syncSchedulerAccountSnapshots(baseCtx, input.AccountIDs)
	return nil
}
