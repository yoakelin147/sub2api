package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrSupplierAccountBatchTooLarge = infraerrors.New(
		http.StatusRequestEntityTooLarge,
		"BATCH_TOO_LARGE",
		"supplier account batch exceeds the maximum size",
	)
	ErrSupplierAccountKindNotAllowed = infraerrors.Forbidden(
		"ACCOUNT_KIND_NOT_ALLOWED",
		"supplier is not allowed to submit this account kind",
	)
	ErrSupplierAccountInputInvalid = infraerrors.New(
		http.StatusUnprocessableEntity,
		"SUPPLIER_ACCOUNT_INPUT_INVALID",
		"supplier account input is invalid",
	)
)

type CreateSupplierAccountInput struct {
	ExternalID  string
	Name        string
	Notes       *string
	Platform    string
	Type        string
	Credentials map[string]any
	ExpiresAt   *time.Time
}

type UpdateSupplierAccountInput struct {
	ExternalID  *string
	Name        *string
	Notes       *string
	Credentials *map[string]any
	ExpiresAt   *time.Time
	Status      *string
}

type SupplierAccountService struct {
	repo      SupplierAccountRepository
	suppliers *SupplierService
	cfg       *config.Config
}

const MaxSupplierAccountBatchSize = 500

type SupplierAccountBatchItemResult struct {
	Index      int    `json:"index"`
	ExternalID string `json:"external_id,omitempty"`
	AccountID  int64  `json:"account_id,omitempty"`
	Success    bool   `json:"success"`
	ErrorCode  string `json:"error_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type SupplierAccountBatchResult struct {
	Total     int                              `json:"total"`
	Succeeded int                              `json:"succeeded"`
	Failed    int                              `json:"failed"`
	Results   []SupplierAccountBatchItemResult `json:"results"`
}

func NewSupplierAccountService(
	repo SupplierAccountRepository,
	suppliers *SupplierService,
	cfg *config.Config,
) *SupplierAccountService {
	return &SupplierAccountService{repo: repo, suppliers: suppliers, cfg: cfg}
}

func (s *SupplierAccountService) Create(ctx context.Context, supplierID int64, input CreateSupplierAccountInput) (*Account, error) {
	supplier, err := s.activeSupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	return s.createWithSupplier(ctx, supplier, input)
}

func (s *SupplierAccountService) createWithSupplier(ctx context.Context, supplier *Supplier, input CreateSupplierAccountInput) (*Account, error) {
	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	accountType := strings.ToLower(strings.TrimSpace(input.Type))
	if !supplierAllowsAccountKind(supplier, platform, accountType) {
		return nil, ErrSupplierAccountKindNotAllowed
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 100 {
		return nil, ErrSupplierAccountInputInvalid
	}
	externalID := strings.TrimSpace(input.ExternalID)
	if len(externalID) > 191 {
		return nil, ErrSupplierAccountInputInvalid
	}
	credentials, err := ValidateSupplierAccountCredentials(s.cfg, platform, accountType, input.Credentials)
	if err != nil {
		return nil, err
	}
	rateMultiplier := 1.0
	account := &Account{
		Name:               name,
		Notes:              normalizeAccountNotes(input.Notes),
		Platform:           platform,
		Type:               accountType,
		Credentials:        SanitizeStoredCredentials(platform, credentials),
		Extra:              map[string]any{},
		Concurrency:        normalizeAccountConcurrency(platform, accountType, 1),
		Priority:           50,
		RateMultiplier:     &rateMultiplier,
		Status:             StatusDisabled,
		Schedulable:        false,
		ReviewStatus:       AccountReviewStatusPending,
		ExpiresAt:          input.ExpiresAt,
		AutoPauseOnExpired: true,
	}
	if externalID != "" {
		account.SupplierExternalID = &externalID
	}
	if err := s.repo.CreateOwned(ctx, supplier.ID, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *SupplierAccountService) BatchCreate(ctx context.Context, supplierID int64, inputs []CreateSupplierAccountInput) (*SupplierAccountBatchResult, error) {
	if len(inputs) == 0 || len(inputs) > MaxSupplierAccountBatchSize {
		return nil, ErrSupplierAccountBatchTooLarge
	}
	supplier, err := s.activeSupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	result := &SupplierAccountBatchResult{Total: len(inputs), Results: make([]SupplierAccountBatchItemResult, 0, len(inputs))}
	for index, input := range inputs {
		item := SupplierAccountBatchItemResult{Index: index, ExternalID: strings.TrimSpace(input.ExternalID)}
		account, err := s.createWithSupplier(ctx, supplier, input)
		if err != nil {
			if infraerrors.Code(err) >= http.StatusInternalServerError {
				return nil, err
			}
			item.ErrorCode = infraerrors.Reason(err)
			item.Message = infraerrors.Message(err)
			result.Failed++
		} else {
			item.Success = true
			item.AccountID = account.ID
			result.Succeeded++
		}
		result.Results = append(result.Results, item)
	}
	return result, nil
}

func (s *SupplierAccountService) GetByID(ctx context.Context, supplierID, accountID int64) (*Account, error) {
	if _, err := s.activeSupplier(ctx, supplierID); err != nil {
		return nil, err
	}
	return s.repo.GetOwnedByID(ctx, supplierID, accountID)
}

func (s *SupplierAccountService) List(ctx context.Context, supplierID int64, params pagination.PaginationParams, filters SupplierAccountFilters) ([]Account, *pagination.PaginationResult, error) {
	if _, err := s.activeSupplier(ctx, supplierID); err != nil {
		return nil, nil, err
	}
	return s.repo.ListOwned(ctx, supplierID, params, filters)
}

func (s *SupplierAccountService) Delete(ctx context.Context, supplierID, accountID int64) error {
	if _, err := s.activeSupplier(ctx, supplierID); err != nil {
		return err
	}
	return s.repo.DeleteOwned(ctx, supplierID, accountID)
}

func (s *SupplierAccountService) Update(ctx context.Context, supplierID, accountID int64, input UpdateSupplierAccountInput) (*Account, error) {
	if _, err := s.activeSupplier(ctx, supplierID); err != nil {
		return nil, err
	}
	account, err := s.repo.GetOwnedByID(ctx, supplierID, accountID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return nil, ErrSupplierAccountInputInvalid
		}
		account.Name = name
	}
	if input.Notes != nil {
		account.Notes = normalizeAccountNotes(input.Notes)
	}
	if input.ExternalID != nil {
		externalID := strings.TrimSpace(*input.ExternalID)
		if len(externalID) > 191 {
			return nil, ErrSupplierAccountInputInvalid
		}
		account.SupplierExternalID = nil
		if externalID != "" {
			account.SupplierExternalID = &externalID
		}
	}
	if input.Credentials != nil {
		credentials, err := ValidateSupplierAccountCredentials(s.cfg, account.Platform, account.Type, *input.Credentials)
		if err != nil {
			return nil, err
		}
		account.Credentials = SanitizeStoredCredentials(account.Platform, credentials)
		account.ReviewStatus = AccountReviewStatusPending
		account.Status = StatusDisabled
		account.Schedulable = false
		account.ReviewedAt = nil
		account.ReviewedBy = nil
		account.ReviewNote = nil
	}
	if input.ExpiresAt != nil {
		account.ExpiresAt = input.ExpiresAt
	}
	if input.Status != nil {
		switch *input.Status {
		case StatusDisabled:
			account.Status = StatusDisabled
			account.Schedulable = false
		case StatusActive:
			if account.ReviewStatus != AccountReviewStatusApproved {
				return nil, ErrSupplierAccountInputInvalid
			}
			account.Status = StatusActive
			account.Schedulable = true
		default:
			return nil, ErrSupplierAccountInputInvalid
		}
	}
	if err := s.repo.UpdateOwned(ctx, supplierID, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *SupplierAccountService) activeSupplier(ctx context.Context, supplierID int64) (*Supplier, error) {
	supplier, err := s.suppliers.GetByID(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	if supplier.Status != domain.SupplierStatusActive {
		return nil, ErrSupplierDisabled
	}
	return supplier, nil
}

func supplierAllowsAccountKind(supplier *Supplier, platform, accountType string) bool {
	for _, kind := range supplier.AllowedAccountKinds {
		if kind.Platform == platform && kind.Type == accountType {
			return true
		}
	}
	return false
}
