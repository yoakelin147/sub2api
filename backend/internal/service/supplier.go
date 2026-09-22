package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrSupplierNotFound = infraerrors.NotFound("SUPPLIER_NOT_FOUND", "supplier not found")
	ErrSupplierExists   = infraerrors.Conflict(
		"SUPPLIER_CODE_EXISTS",
		"supplier code already exists",
	)
	ErrSupplierAccountKindInvalid = infraerrors.BadRequest(
		"SUPPLIER_ACCOUNT_KIND_INVALID",
		"supplier account kind is invalid",
	)
	ErrSupplierCodeInvalid   = infraerrors.BadRequest("SUPPLIER_CODE_INVALID", "supplier code is invalid")
	ErrSupplierNameInvalid   = infraerrors.BadRequest("SUPPLIER_NAME_INVALID", "supplier name is invalid")
	ErrSupplierStatusInvalid = infraerrors.BadRequest(
		"SUPPLIER_STATUS_INVALID",
		"supplier status is invalid",
	)
)

var supplierCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type SupplierAccountKind = domain.SupplierAccountKind

type Supplier struct {
	ID                  int64
	Code                string
	Name                string
	Status              string
	Notes               *string
	AllowedAccountKinds []SupplierAccountKind
	TokenSelector       *string
	TokenHash           *string
	TokenPrefix         *string
	TokenCreatedAt      *time.Time
	TokenLastUsedAt     *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
}

type SupplierListFilters struct {
	Status string
	Search string
}

type SupplierRepository interface {
	Create(ctx context.Context, supplier *Supplier) error
	GetByID(ctx context.Context, id int64) (*Supplier, error)
	GetByTokenSelector(ctx context.Context, selector string) (*Supplier, error)
	Update(ctx context.Context, supplier *Supplier) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params pagination.PaginationParams, filters SupplierListFilters) ([]Supplier, *pagination.PaginationResult, error)
	UpdateTokenLastUsed(ctx context.Context, id int64, usedAt time.Time) error
}

type CreateSupplierInput struct {
	Code                string
	Name                string
	Status              string
	Notes               *string
	AllowedAccountKinds []SupplierAccountKind
}

type UpdateSupplierInput struct {
	Name                *string
	Status              *string
	Notes               *string
	AllowedAccountKinds *[]SupplierAccountKind
}

type SupplierService struct {
	repo SupplierRepository
}

func NewSupplierService(repo SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

func (s *SupplierService) Create(ctx context.Context, input CreateSupplierInput) (*Supplier, error) {
	code := strings.ToLower(strings.TrimSpace(input.Code))
	if !supplierCodePattern.MatchString(code) {
		return nil, ErrSupplierCodeInvalid
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 120 {
		return nil, ErrSupplierNameInvalid
	}
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status == "" {
		status = domain.SupplierStatusActive
	}
	if !validSupplierStatus(status) {
		return nil, ErrSupplierStatusInvalid
	}
	kinds, err := NormalizeSupplierAccountKinds(input.AllowedAccountKinds)
	if err != nil {
		return nil, err
	}
	supplier := &Supplier{
		Code:                code,
		Name:                name,
		Status:              status,
		Notes:               normalizeSupplierNotes(input.Notes),
		AllowedAccountKinds: kinds,
	}
	if err := s.repo.Create(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *SupplierService) GetByID(ctx context.Context, id int64) (*Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SupplierService) List(
	ctx context.Context,
	params pagination.PaginationParams,
	filters SupplierListFilters,
) ([]Supplier, *pagination.PaginationResult, error) {
	return s.repo.List(ctx, params, filters)
}

func (s *SupplierService) Update(ctx context.Context, id int64, input UpdateSupplierInput) (*Supplier, error) {
	supplier, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 120 {
			return nil, ErrSupplierNameInvalid
		}
		supplier.Name = name
	}
	if input.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*input.Status))
		if !validSupplierStatus(status) {
			return nil, ErrSupplierStatusInvalid
		}
		supplier.Status = status
	}
	if input.Notes != nil {
		supplier.Notes = normalizeSupplierNotes(input.Notes)
	}
	if input.AllowedAccountKinds != nil {
		kinds, err := NormalizeSupplierAccountKinds(*input.AllowedAccountKinds)
		if err != nil {
			return nil, err
		}
		supplier.AllowedAccountKinds = kinds
	}
	if err := s.repo.Update(ctx, supplier); err != nil {
		return nil, err
	}
	return supplier, nil
}

func (s *SupplierService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func validSupplierStatus(status string) bool {
	return status == domain.SupplierStatusActive || status == domain.SupplierStatusDisabled
}

func normalizeSupplierNotes(notes *string) *string {
	if notes == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*notes)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

var supplierAccountKinds = map[string]map[string]struct{}{
	PlatformOpenAI: {
		AccountTypeAPIKey: {}, AccountTypeUpstream: {}, AccountTypeOAuth: {}, AccountTypeSetupToken: {},
	},
	PlatformAnthropic: {
		AccountTypeAPIKey: {}, AccountTypeUpstream: {}, AccountTypeOAuth: {}, AccountTypeSetupToken: {},
		AccountTypeBedrock: {}, AccountTypeServiceAccount: {},
	},
	PlatformGemini: {
		AccountTypeAPIKey: {}, AccountTypeOAuth: {}, AccountTypeServiceAccount: {},
	},
	PlatformAntigravity: {
		AccountTypeAPIKey: {}, AccountTypeOAuth: {},
	},
	PlatformGrok: {
		AccountTypeAPIKey: {}, AccountTypeOAuth: {},
	},
	PlatformKimi:       {AccountTypeAPIKey: {}},
	PlatformZhipu:      {AccountTypeAPIKey: {}},
	PlatformDeepseek:   {AccountTypeAPIKey: {}},
	PlatformMiniMax:    {AccountTypeAPIKey: {}},
	PlatformOpenCodeGo: {AccountTypeAPIKey: {}},
}

func NormalizeSupplierAccountKinds(kinds []SupplierAccountKind) ([]SupplierAccountKind, error) {
	result := make([]SupplierAccountKind, 0, len(kinds))
	seen := make(map[SupplierAccountKind]struct{}, len(kinds))
	for _, kind := range kinds {
		kind.Platform = strings.ToLower(strings.TrimSpace(kind.Platform))
		kind.Type = strings.ToLower(strings.TrimSpace(kind.Type))
		types, ok := supplierAccountKinds[kind.Platform]
		if !ok {
			return nil, ErrSupplierAccountKindInvalid
		}
		if _, ok := types[kind.Type]; !ok {
			return nil, ErrSupplierAccountKindInvalid
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		result = append(result, kind)
	}
	return result, nil
}
