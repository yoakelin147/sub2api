package repository

import (
	"context"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/supplier"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *supplierRepository) GetStatsBySupplierIDs(ctx context.Context, supplierIDs []int64) (map[int64]service.SupplierStats, error) {
	stats := make(map[int64]service.SupplierStats, len(supplierIDs))
	if len(supplierIDs) == 0 {
		return stats, nil
	}
	rows, err := r.client.QueryContext(ctx, `
		SELECT s.id,
			(SELECT COUNT(*) FROM users u WHERE u.supplier_id = s.id AND u.deleted_at IS NULL),
			(SELECT COUNT(*) FROM accounts a WHERE a.supplier_id = s.id AND a.deleted_at IS NULL),
			(SELECT COUNT(*) FROM accounts a WHERE a.supplier_id = s.id AND a.deleted_at IS NULL AND a.review_status = 'pending'),
			(SELECT COUNT(*) FROM accounts a WHERE a.supplier_id = s.id AND a.deleted_at IS NULL AND a.schedulable = true),
			(SELECT COUNT(*) FROM accounts a WHERE a.supplier_id = s.id AND a.deleted_at IS NULL AND COALESCE(a.error_message, '') <> '')
		FROM suppliers s
		WHERE s.id = ANY($1) AND s.deleted_at IS NULL
	`, pq.Array(supplierIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var supplierID int64
		var item service.SupplierStats
		if err := rows.Scan(&supplierID, &item.MemberCount, &item.AccountCount, &item.PendingCount, &item.SchedulableCount, &item.ErrorCount); err != nil {
			return nil, err
		}
		stats[supplierID] = item
	}
	return stats, rows.Err()
}

type supplierRepository struct {
	client *dbent.Client
}

func NewSupplierRepository(client *dbent.Client) service.SupplierRepository {
	return &supplierRepository{client: client}
}

func (r *supplierRepository) Create(ctx context.Context, model *service.Supplier) error {
	builder := r.client.Supplier.Create().
		SetCode(model.Code).
		SetName(model.Name).
		SetStatus(model.Status).
		SetAllowedAccountKinds(model.AllowedAccountKinds).
		SetReviewRequired(model.ReviewRequired).
		SetAutoApproveGroups(model.AutoApproveGroups).
		SetNillableNotes(model.Notes).
		SetNillableTokenSelector(model.TokenSelector).
		SetNillableTokenHash(model.TokenHash).
		SetNillableTokenPrefix(model.TokenPrefix).
		SetNillableTokenCreatedAt(model.TokenCreatedAt).
		SetNillableTokenLastUsedAt(model.TokenLastUsedAt)

	created, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrSupplierExists)
	}
	applySupplierEntityToService(model, created)
	return nil
}

func (r *supplierRepository) GetByID(ctx context.Context, id int64) (*service.Supplier, error) {
	row, err := r.client.Supplier.Query().Where(supplier.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupplierNotFound, nil)
	}
	return supplierEntityToService(row), nil
}

func (r *supplierRepository) GetByTokenSelector(ctx context.Context, selector string) (*service.Supplier, error) {
	row, err := r.client.Supplier.Query().
		Where(supplier.TokenSelectorEQ(strings.TrimSpace(selector))).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupplierNotFound, nil)
	}
	return supplierEntityToService(row), nil
}

func (r *supplierRepository) Update(ctx context.Context, model *service.Supplier) error {
	builder := r.client.Supplier.UpdateOneID(model.ID).
		SetCode(model.Code).
		SetName(model.Name).
		SetStatus(model.Status).
		SetAllowedAccountKinds(model.AllowedAccountKinds).
		SetReviewRequired(model.ReviewRequired).
		SetAutoApproveGroups(model.AutoApproveGroups).
		SetNillableNotes(model.Notes)

	if model.Notes == nil {
		builder.ClearNotes()
	}
	if model.TokenSelector == nil {
		builder.ClearTokenSelector().ClearTokenHash().ClearTokenPrefix().ClearTokenCreatedAt().ClearTokenLastUsedAt()
	} else {
		builder.
			SetNillableTokenSelector(model.TokenSelector).
			SetNillableTokenHash(model.TokenHash).
			SetNillableTokenPrefix(model.TokenPrefix).
			SetNillableTokenCreatedAt(model.TokenCreatedAt).
			SetNillableTokenLastUsedAt(model.TokenLastUsedAt)
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrSupplierNotFound, service.ErrSupplierExists)
	}
	applySupplierEntityToService(model, updated)
	return nil
}

func (r *supplierRepository) Delete(ctx context.Context, id int64) error {
	return translatePersistenceError(
		r.client.Supplier.DeleteOneID(id).Exec(ctx),
		service.ErrSupplierNotFound,
		nil,
	)
}

func (r *supplierRepository) List(
	ctx context.Context,
	params pagination.PaginationParams,
	filters service.SupplierListFilters,
) ([]service.Supplier, *pagination.PaginationResult, error) {
	query := r.client.Supplier.Query()
	if filters.Status != "" {
		query = query.Where(supplier.StatusEQ(filters.Status))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		query = query.Where(supplier.Or(supplier.CodeContainsFold(search), supplier.NameContainsFold(search)))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := query.
		Order(dbent.Desc(supplier.FieldCreatedAt), dbent.Desc(supplier.FieldID)).
		Offset(params.Offset()).
		Limit(params.Limit()).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	items := make([]service.Supplier, 0, len(rows))
	for _, row := range rows {
		items = append(items, *supplierEntityToService(row))
	}
	return items, paginationResultFromTotal(int64(total), params), nil
}

func (r *supplierRepository) UpdateTokenLastUsed(ctx context.Context, id int64, usedAt time.Time) error {
	_, err := r.client.Supplier.UpdateOneID(id).SetTokenLastUsedAt(usedAt).Save(ctx)
	return translatePersistenceError(err, service.ErrSupplierNotFound, nil)
}

func supplierEntityToService(row *dbent.Supplier) *service.Supplier {
	if row == nil {
		return nil
	}
	model := &service.Supplier{}
	applySupplierEntityToService(model, row)
	return model
}

func applySupplierEntityToService(model *service.Supplier, row *dbent.Supplier) {
	*model = service.Supplier{
		ID:                  row.ID,
		Code:                row.Code,
		Name:                row.Name,
		Status:              row.Status,
		Notes:               row.Notes,
		AllowedAccountKinds: append([]service.SupplierAccountKind{}, row.AllowedAccountKinds...),
		ReviewRequired:      row.ReviewRequired,
		AutoApproveGroups:   row.AutoApproveGroups,
		TokenSelector:       row.TokenSelector,
		TokenHash:           row.TokenHash,
		TokenPrefix:         row.TokenPrefix,
		TokenCreatedAt:      row.TokenCreatedAt,
		TokenLastUsedAt:     row.TokenLastUsedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
		DeletedAt:           row.DeletedAt,
	}
}
