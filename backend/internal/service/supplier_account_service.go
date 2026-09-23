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
	ErrSupplierProxyRequired              = infraerrors.New(http.StatusUnprocessableEntity, "SUPPLIER_PROXY_REQUIRED", "an active platform proxy is required")
	ErrSupplierAutoApproveGroupRequired   = infraerrors.New(http.StatusUnprocessableEntity, "SUPPLIER_AUTO_APPROVE_GROUP_REQUIRED", "an active platform group must be configured for automatic approval")
	ErrSupplierPasswordEncryptionRequired = infraerrors.New(http.StatusUnprocessableEntity, "SUPPLIER_PASSWORD_ENCRYPTION_REQUIRED", "configure a fixed TOTP_ENCRYPTION_KEY before accepting supplier login passwords")
)

type CreateSupplierAccountInput struct {
	ExternalID         string
	Name               string
	Notes              *string
	Platform           string
	Type               string
	Credentials        map[string]any
	ExpiresAt          *time.Time
	ProxyID            *int64
	Concurrency        *int
	Priority           *int
	LoadFactor         *int
	AutoPauseOnExpired *bool
}

type UpdateSupplierAccountInput struct {
	ExternalID         *string
	Name               *string
	Notes              *string
	Credentials        *map[string]any
	ExpiresAt          *time.Time
	Status             *string
	ProxyID            *int64
	Concurrency        *int
	Priority           *int
	LoadFactor         *int
	AutoPauseOnExpired *bool
}

type SupplierProxyOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type SupplierAccountService struct {
	repo      SupplierAccountRepository
	suppliers *SupplierService
	groups    GroupRepository
	proxies   *ProxyService
	cfg       *config.Config
	encryptor SecretEncryptor
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
	groups GroupRepository,
	proxies *ProxyService,
	cfg *config.Config,
	encryptor SecretEncryptor,
) *SupplierAccountService {
	return &SupplierAccountService{repo: repo, suppliers: suppliers, groups: groups, proxies: proxies, cfg: cfg, encryptor: encryptor}
}

func (s *SupplierAccountService) secureLoginPassword(credentials map[string]any) error {
	password, ok := credentials["password"].(string)
	if !ok {
		return nil
	}
	if s.encryptor == nil || s.cfg == nil || !s.cfg.Totp.EncryptionKeyConfigured {
		return ErrSupplierPasswordEncryptionRequired
	}
	encrypted, err := s.encryptor.Encrypt(password)
	if err != nil {
		return err
	}
	delete(credentials, "password")
	credentials["login_password_encrypted"] = encrypted
	return nil
}

func (s *SupplierAccountService) AvailableProxies(ctx context.Context, supplierID int64) ([]SupplierProxyOption, error) {
	if _, err := s.activeSupplier(ctx, supplierID); err != nil {
		return nil, err
	}
	if s.proxies == nil {
		return nil, ErrSupplierProxyRequired
	}
	proxies, err := s.proxies.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]SupplierProxyOption, 0, len(proxies))
	for _, proxy := range proxies {
		if !proxy.IsExpired(time.Now()) {
			options = append(options, SupplierProxyOption{ID: proxy.ID, Name: proxy.Name})
		}
	}
	return options, nil
}

func (s *SupplierAccountService) AuthorizeOAuth(ctx context.Context, supplierID int64, platform, accountType string) error {
	supplier, err := s.activeSupplier(ctx, supplierID)
	if err != nil {
		return err
	}
	if !supplierAllowsAccountKind(supplier, platform, accountType) {
		return ErrSupplierAccountKindNotAllowed
	}
	return nil
}

func (s *SupplierAccountService) validateProxy(ctx context.Context, proxyID *int64) error {
	if proxyID == nil || *proxyID <= 0 || s.proxies == nil {
		return ErrSupplierProxyRequired
	}
	proxy, err := s.proxies.GetByID(ctx, *proxyID)
	if err != nil || proxy == nil || !proxy.IsActive() || proxy.IsExpired(time.Now()) {
		return ErrSupplierProxyRequired
	}
	return nil
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
	if err := s.secureLoginPassword(credentials); err != nil {
		return nil, err
	}
	if err := s.validateProxy(ctx, input.ProxyID); err != nil {
		return nil, err
	}
	if input.Concurrency != nil && (*input.Concurrency < 1 || *input.Concurrency > 10000) {
		return nil, ErrSupplierAccountInputInvalid
	}
	if input.Priority != nil && (*input.Priority < 0 || *input.Priority > 10000) {
		return nil, ErrSupplierAccountInputInvalid
	}
	if input.LoadFactor != nil && (*input.LoadFactor < 1 || *input.LoadFactor > 10000) {
		return nil, ErrSupplierAccountInputInvalid
	}
	groups := []AccountGroup(nil)
	if !supplier.ReviewRequired {
		groupID := supplier.AutoApproveGroups[platform]
		if groupID <= 0 || s.groups == nil {
			return nil, ErrSupplierAutoApproveGroupRequired
		}
		group, err := s.groups.GetByID(ctx, groupID)
		if err != nil || group == nil || group.Status != StatusActive || group.Platform != platform || (group.RequireOAuthOnly && accountType != AccountTypeOAuth) {
			return nil, ErrSupplierAutoApproveGroupRequired
		}
		groups = []AccountGroup{{GroupID: groupID, Priority: 1}}
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
		ProxyID:            input.ProxyID,
	}
	if input.Concurrency != nil {
		account.Concurrency = normalizeAccountConcurrency(platform, accountType, *input.Concurrency)
	}
	if input.Priority != nil {
		account.Priority = *input.Priority
	}
	if input.LoadFactor != nil {
		account.LoadFactor = input.LoadFactor
	}
	if input.AutoPauseOnExpired != nil {
		account.AutoPauseOnExpired = *input.AutoPauseOnExpired
	}
	if !supplier.ReviewRequired {
		account.ReviewStatus = AccountReviewStatusApproved
		account.Status = StatusActive
		account.Schedulable = true
		now := time.Now().UTC()
		account.ReviewedAt = &now
	}
	if externalID != "" {
		account.SupplierExternalID = &externalID
	}
	if err := s.repo.CreateOwned(ctx, supplier.ID, account, groups); err != nil {
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

func (s *SupplierAccountService) ListForAdmin(ctx context.Context, supplierID int64, params pagination.PaginationParams, filters SupplierAccountFilters) ([]Account, *pagination.PaginationResult, error) {
	if _, err := s.suppliers.GetByID(ctx, supplierID); err != nil {
		return nil, nil, err
	}
	return s.repo.ListOwned(ctx, supplierID, params, filters)
}

func (s *SupplierAccountService) GetForAdmin(ctx context.Context, supplierID, accountID int64) (*Account, error) {
	if _, err := s.suppliers.GetByID(ctx, supplierID); err != nil {
		return nil, err
	}
	return s.repo.GetOwnedByID(ctx, supplierID, accountID)
}

func (s *SupplierAccountService) RevealLoginPassword(ctx context.Context, supplierID, accountID int64) (string, error) {
	account, err := s.GetForAdmin(ctx, supplierID, accountID)
	if err != nil {
		return "", err
	}
	encrypted, ok := account.Credentials["login_password_encrypted"].(string)
	if !ok || encrypted == "" {
		return "", ErrSupplierCredentialsInvalid
	}
	if s.encryptor == nil {
		return "", ErrSupplierPasswordEncryptionRequired
	}
	return s.encryptor.Decrypt(encrypted)
}

func (s *SupplierAccountService) Delete(ctx context.Context, supplierID, accountID int64) error {
	if _, err := s.activeSupplier(ctx, supplierID); err != nil {
		return err
	}
	return s.repo.DeleteOwned(ctx, supplierID, accountID)
}

func (s *SupplierAccountService) Update(ctx context.Context, supplierID, accountID int64, input UpdateSupplierAccountInput) (*Account, error) {
	supplier, err := s.activeSupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	account, err := s.repo.GetOwnedByID(ctx, supplierID, accountID)
	if err != nil {
		return nil, err
	}
	proxyID := account.ProxyID
	proxyChanged := input.ProxyID != nil && (proxyID == nil || *proxyID != *input.ProxyID)
	if input.ProxyID != nil {
		proxyID = input.ProxyID
	}
	if err := s.validateProxy(ctx, proxyID); err != nil {
		return nil, err
	}
	if input.ProxyID != nil {
		account.ProxyID = input.ProxyID
	}
	if input.Concurrency != nil {
		if *input.Concurrency < 1 || *input.Concurrency > 10000 {
			return nil, ErrSupplierAccountInputInvalid
		}
		account.Concurrency = normalizeAccountConcurrency(account.Platform, account.Type, *input.Concurrency)
	}
	if input.Priority != nil {
		if *input.Priority < 0 || *input.Priority > 10000 {
			return nil, ErrSupplierAccountInputInvalid
		}
		account.Priority = *input.Priority
	}
	if input.LoadFactor != nil {
		if *input.LoadFactor < 1 || *input.LoadFactor > 10000 {
			return nil, ErrSupplierAccountInputInvalid
		}
		account.LoadFactor = input.LoadFactor
	}
	if input.AutoPauseOnExpired != nil {
		account.AutoPauseOnExpired = *input.AutoPauseOnExpired
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
	if input.Credentials != nil || proxyChanged {
		if input.Credentials != nil {
			credentials, err := ValidateSupplierAccountCredentials(s.cfg, account.Platform, account.Type, *input.Credentials)
			if err != nil {
				return nil, err
			}
			if err := s.secureLoginPassword(credentials); err != nil {
				return nil, err
			}
			account.Credentials = SanitizeStoredCredentials(account.Platform, credentials)
		}
		if supplier.ReviewRequired {
			account.ReviewStatus = AccountReviewStatusPending
			account.Status = StatusDisabled
			account.Schedulable = false
			account.ReviewedAt = nil
			account.ReviewedBy = nil
			account.ReviewNote = nil
		}
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

func (s *SupplierAccountService) Review(ctx context.Context, supplierID int64, input SupplierAccountReviewInput) error {
	ids := uniquePositiveInt64s(input.AccountIDs)
	if supplierID <= 0 || len(ids) == 0 {
		return ErrSupplierAccountInputInvalid
	}
	input.AccountIDs = ids
	switch input.Action {
	case SupplierAccountReviewApprove:
		if len(input.GroupIDs) == 0 || s.groups == nil {
			return ErrSupplierAccountInputInvalid
		}
		for _, groupID := range uniquePositiveInt64s(input.GroupIDs) {
			if _, err := s.groups.GetByID(ctx, groupID); err != nil {
				return err
			}
		}
		input.GroupIDs = uniquePositiveInt64s(input.GroupIDs)
	case SupplierAccountReviewReject, SupplierAccountReviewPause:
		input.GroupIDs = nil
	default:
		return ErrSupplierAccountInputInvalid
	}
	accounts, err := s.repo.GetOwnedByIDs(ctx, supplierID, ids)
	if err != nil {
		return err
	}
	if len(accounts) != len(ids) {
		return ErrSupplierAccountNotFound
	}
	if input.Action == SupplierAccountReviewApprove {
		for _, account := range accounts {
			if err := s.validateProxy(ctx, account.ProxyID); err != nil {
				return err
			}
			for _, groupID := range input.GroupIDs {
				group, err := s.groups.GetByID(ctx, groupID)
				if err != nil || group == nil || group.Status != StatusActive || group.Platform != account.Platform || (group.RequireOAuthOnly && account.Type != AccountTypeOAuth) {
					return ErrSupplierAccountInputInvalid
				}
			}
		}
	}
	return s.repo.ReviewOwned(ctx, supplierID, input)
}

func uniquePositiveInt64s(values []int64) []int64 {
	result := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
