package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"math"
	"net/http"
	"slices"
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
	ErrSupplierAccountTestNotApproved = infraerrors.Forbidden(
		"SUPPLIER_ACCOUNT_TEST_NOT_APPROVED",
		"supplier account must be approved before testing",
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
	Extra              map[string]any
	ExpiresAt          *time.Time
	ProxyID            *int64
	Concurrency        *int
	Priority           *int
	LoadFactor         *int
	AutoPauseOnExpired *bool
	GroupIDs           []int64
}

type UpdateSupplierAccountInput struct {
	ExternalID         *string
	Name               *string
	Notes              *string
	Credentials        *map[string]any
	Extra              *map[string]any
	ExpiresAt          *time.Time
	ClearExpiresAt     bool
	Status             *string
	ProxyID            *int64
	Concurrency        *int
	Priority           *int
	LoadFactor         *int
	ClearLoadFactor    bool
	RateMultiplier     *float64
	AutoPauseOnExpired *bool
	GroupIDs           []int64
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

type SupplierGroupOption struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Platform         string `json:"platform"`
	RequireOAuthOnly bool   `json:"require_oauth_only"`
}

func (s *SupplierAccountService) AuthorizedGroups(ctx context.Context, supplier *Supplier) ([]SupplierGroupOption, error) {
	options := []SupplierGroupOption{}
	if supplier.ReviewRequired || s.groups == nil {
		return options, nil
	}
	for platform, ids := range supplier.AutoApproveGroups {
		for _, id := range ids {
			group, err := s.groups.GetByID(ctx, id)
			if err != nil {
				if errors.Is(err, ErrGroupNotFound) {
					continue
				}
				return nil, err
			}
			if group != nil && group.Status == StatusActive && group.Platform == platform {
				options = append(options, SupplierGroupOption{ID: group.ID, Name: group.Name, Description: group.Description, Platform: platform, RequireOAuthOnly: group.RequireOAuthOnly})
			}
		}
	}
	slices.SortFunc(options, func(first, second SupplierGroupOption) int {
		if comparison := strings.Compare(first.Platform, second.Platform); comparison != 0 {
			return comparison
		}
		return cmp.Compare(first.ID, second.ID)
	})
	return options, nil
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
	if s.proxies == nil {
		return ErrSupplierProxyRequired
	}
	if proxyID == nil || *proxyID == 0 {
		proxies, err := s.proxies.ListActive(ctx)
		if err != nil {
			return err
		}
		for _, proxy := range proxies {
			if !proxy.IsExpired(time.Now()) {
				return ErrSupplierProxyRequired
			}
		}
		return nil
	}
	if *proxyID < 0 {
		return ErrSupplierProxyRequired
	}
	proxy, err := s.proxies.GetByID(ctx, *proxyID)
	if err != nil || proxy == nil || !proxy.IsActive() || proxy.IsExpired(time.Now()) {
		return ErrSupplierProxyRequired
	}
	return nil
}

func normalizeSupplierProxyID(proxyID *int64) *int64 {
	if proxyID == nil || *proxyID == 0 {
		return nil
	}
	return proxyID
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
	if supplier.ReviewRequired && (input.Notes != nil || input.ExpiresAt != nil || input.Concurrency != nil || input.Priority != nil || input.LoadFactor != nil || input.AutoPauseOnExpired != nil || input.GroupIDs != nil || input.Extra != nil) {
		return nil, ErrSupplierAccountInputInvalid
	}
	if err := validateSupplierExtra(platform, accountType, input.Extra); err != nil {
		return nil, err
	}
	if err := validateSupplierCredentialPermissions(platform, accountType, input.Credentials, supplier.ReviewRequired); err != nil {
		return nil, err
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
		groups, err = s.authorizedGroup(ctx, supplier, platform, accountType, input.GroupIDs)
		if err != nil {
			return nil, err
		}
	}
	rateMultiplier := 1.0
	account := &Account{
		Name:               name,
		Notes:              normalizeAccountNotes(input.Notes),
		Platform:           platform,
		Type:               accountType,
		Credentials:        SanitizeStoredCredentials(platform, credentials),
		Extra:              input.Extra,
		Concurrency:        normalizeAccountConcurrency(platform, accountType, 1),
		Priority:           50,
		RateMultiplier:     &rateMultiplier,
		Status:             StatusDisabled,
		Schedulable:        false,
		ReviewStatus:       AccountReviewStatusPending,
		ExpiresAt:          input.ExpiresAt,
		AutoPauseOnExpired: true,
		ProxyID:            normalizeSupplierProxyID(input.ProxyID),
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
	return s.updateOwned(ctx, supplierID, accountID, input, supplier, false)
}

func (s *SupplierAccountService) UpdateForAdmin(ctx context.Context, supplierID, accountID int64, input UpdateSupplierAccountInput) (*Account, error) {
	if input.Status != nil {
		return nil, ErrSupplierAccountInputInvalid
	}
	supplier, err := s.suppliers.GetByID(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	return s.updateOwned(ctx, supplierID, accountID, input, supplier, true)
}

func (s *SupplierAccountService) updateOwned(ctx context.Context, supplierID, accountID int64, input UpdateSupplierAccountInput, supplier *Supplier, admin bool) (*Account, error) {
	account, err := s.repo.GetOwnedByID(ctx, supplierID, accountID)
	if err != nil {
		return nil, err
	}
	if !admin && (input.RateMultiplier != nil || input.ClearExpiresAt || input.ClearLoadFactor || supplier.ReviewRequired && (input.Notes != nil || input.ExpiresAt != nil || input.Concurrency != nil || input.Priority != nil || input.LoadFactor != nil || input.AutoPauseOnExpired != nil || input.Extra != nil || input.GroupIDs != nil)) || admin && input.GroupIDs != nil {
		return nil, ErrSupplierAccountInputInvalid
	}
	var groups []AccountGroup
	if !admin && !supplier.ReviewRequired && input.GroupIDs != nil {
		groups, err = s.authorizedGroup(ctx, supplier, account.Platform, account.Type, input.GroupIDs)
		if err != nil {
			return nil, err
		}
	}
	if input.Extra != nil {
		if admin {
			encoded, err := json.Marshal(*input.Extra)
			if err != nil || len(encoded) > maxSupplierCredentialBytes {
				return nil, ErrSupplierAccountInputInvalid
			}
			if err := ValidateOpenAILongContextBillingExtra(account.Platform, *input.Extra); err != nil {
				return nil, ErrSupplierAccountInputInvalid.WithCause(err)
			}
			if err := ValidateUpstreamRequestIDHeaderExtra(*input.Extra); err != nil {
				return nil, ErrSupplierAccountInputInvalid.WithCause(err)
			}
			for _, key := range []string{OllamaCloudUsageSessionExtraKey, OllamaCloudUsageAutoRefreshExtraKey, OllamaCloudUsageSnapshotExtraKey} {
				if _, exists := (*input.Extra)[key]; !exists {
					if value, stored := account.Extra[key]; stored {
						(*input.Extra)[key] = value
					}
				}
			}
			account.Extra = *input.Extra
		} else {
			if err := validateSupplierExtra(account.Platform, account.Type, *input.Extra); err != nil {
				return nil, err
			}
			if account.Extra == nil {
				account.Extra = make(map[string]any)
			}
			for _, key := range []string{AccountExtraUpstreamRequestIDHeader, "openai_compact_mode", featureKeyWebSearchEmulation} {
				delete(account.Extra, key)
			}
			for key, value := range *input.Extra {
				account.Extra[key] = value
			}
		}
	}
	if !admin && input.Credentials != nil {
		if err := validateSupplierCredentialPermissions(account.Platform, account.Type, *input.Credentials, supplier.ReviewRequired); err != nil {
			return nil, err
		}
	}
	proxyID := account.ProxyID
	proxyChanged := input.ProxyID != nil && (proxyID == nil || *proxyID != *input.ProxyID)
	if input.ProxyID != nil {
		proxyID = normalizeSupplierProxyID(input.ProxyID)
	}
	if err := s.validateProxy(ctx, proxyID); err != nil {
		return nil, err
	}
	if input.ProxyID != nil {
		account.ProxyID = proxyID
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
	if admin && input.ClearLoadFactor {
		account.LoadFactor = nil
	} else if input.LoadFactor != nil {
		if *input.LoadFactor < 1 || *input.LoadFactor > 10000 {
			return nil, ErrSupplierAccountInputInvalid
		}
		account.LoadFactor = input.LoadFactor
	}
	if admin && input.RateMultiplier != nil {
		if math.IsNaN(*input.RateMultiplier) || *input.RateMultiplier < 0 || *input.RateMultiplier > 10000 {
			return nil, ErrSupplierAccountInputInvalid
		}
		account.RateMultiplier = input.RateMultiplier
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
			if admin {
				credentials, err := s.adminCredentials(account, *input.Credentials)
				if err != nil {
					return nil, err
				}
				account.Credentials = credentials
			} else {
				configurationOnly := !supplier.ReviewRequired && len(*input.Credentials) > 0
				for key := range *input.Credentials {
					if key != "model_mapping" && key != "compact_model_mapping" && key != "protocol_rules" {
						configurationOnly = false
					}
				}
				if configurationOnly {
					if len(account.Credentials) == 0 {
						return nil, ErrSupplierCredentialsInvalid
					}
					encoded, err := json.Marshal(*input.Credentials)
					if err != nil || len(encoded) > maxSupplierCredentialBytes {
						return nil, ErrSupplierCredentialsInvalid
					}
					credentials := maps.Clone(account.Credentials)
					for key, value := range *input.Credentials {
						if key == "protocol_rules" {
							if account.Platform != PlatformOpenCodeGo {
								return nil, ErrSupplierCredentialsInvalid
							}
						} else if err := validateSupplierModelMapping(value); err != nil {
							return nil, err
						}
						credentials[key] = value
					}
					if account.Platform == PlatformOpenCodeGo {
						if err := NormalizeOpenCodeGoProtocolRulesCredentials(credentials); err != nil {
							return nil, ErrSupplierCredentialsInvalid.WithCause(err)
						}
					}
					account.Credentials = credentials
				} else {
					credentials, err := ValidateSupplierAccountCredentials(s.cfg, account.Platform, account.Type, *input.Credentials)
					if err != nil {
						return nil, err
					}
					if err := s.secureLoginPassword(credentials); err != nil {
						return nil, err
					}
					account.Credentials = SanitizeStoredCredentials(account.Platform, credentials)
				}
			}
		}
		if !admin && supplier.ReviewRequired {
			account.ReviewStatus = AccountReviewStatusPending
			account.Status = StatusDisabled
			account.Schedulable = false
			account.ReviewedAt = nil
			account.ReviewedBy = nil
			account.ReviewNote = nil
		}
	}
	if admin && input.ClearExpiresAt {
		account.ExpiresAt = nil
	} else if input.ExpiresAt != nil {
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
	if err := s.repo.UpdateOwned(ctx, supplierID, account, groups); err != nil {
		return nil, err
	}
	if groups != nil {
		account.AccountGroups = groups
		account.GroupIDs = make([]int64, 0, len(groups))
		for _, group := range groups {
			account.GroupIDs = append(account.GroupIDs, group.GroupID)
		}
	}
	return account, nil
}

func (s *SupplierAccountService) authorizedGroup(ctx context.Context, supplier *Supplier, platform, accountType string, requested []int64) ([]AccountGroup, error) {
	allowed := supplier.AutoApproveGroups[platform]
	if len(allowed) == 0 || s.groups == nil {
		return nil, ErrSupplierAutoApproveGroupRequired
	}
	if len(requested) == 0 {
		for _, groupID := range allowed {
			group, err := s.groups.GetByID(ctx, groupID)
			if err == nil && group != nil && group.Status == StatusActive && group.Platform == platform && (!group.RequireOAuthOnly || accountType == AccountTypeOAuth) {
				return []AccountGroup{{GroupID: groupID, Priority: 1}}, nil
			}
		}
		return nil, ErrSupplierAutoApproveGroupRequired
	}
	groups := make([]AccountGroup, 0, len(requested))
	seen := make(map[int64]bool, len(requested))
	for _, groupID := range requested {
		if groupID <= 0 || seen[groupID] || !slices.Contains(allowed, groupID) {
			return nil, ErrSupplierAutoApproveGroupRequired
		}
		seen[groupID] = true
		group, err := s.groups.GetByID(ctx, groupID)
		if err != nil || group == nil || group.Status != StatusActive || group.Platform != platform || group.RequireOAuthOnly && accountType != AccountTypeOAuth {
			return nil, ErrSupplierAutoApproveGroupRequired
		}
		groups = append(groups, AccountGroup{GroupID: groupID, Priority: 1})
	}
	return groups, nil
}

func (s *SupplierAccountService) adminCredentials(account *Account, input map[string]any) (map[string]any, error) {
	encoded, err := json.Marshal(input)
	if err != nil || len(encoded) > maxSupplierCredentialBytes {
		return nil, ErrSupplierCredentialsInvalid
	}
	spec := supplierCredentialSpecs[SupplierAccountKind{Platform: account.Platform, Type: account.Type}]
	for key := range input {
		if IsSensitiveCredentialKey(key) {
			return nil, ErrSupplierCredentialsInvalid
		}
		if key == "model_mapping" || key == "compact_model_mapping" {
			if err := validateSupplierModelMapping(input[key]); err != nil {
				return nil, err
			}
		}
		if _, allowed := spec.allowed[key]; !allowed {
			switch key {
			case "model_mapping", "compact_model_mapping", "pool_mode", "pool_mode_retry_count", "pool_mode_retry_status_codes":
			default:
				if _, exists := account.Credentials[key]; !exists {
					return nil, ErrSupplierCredentialsInvalid
				}
			}
		}
	}
	for key := range spec.required {
		if _, existed := account.Credentials[key]; existed && !IsSensitiveCredentialKey(key) && !nonEmptyCredentialValue(input[key]) {
			return nil, ErrSupplierCredentialsInvalid
		}
	}
	credentials := MergePreservingSensitiveCreds(account.Credentials, input)
	if account.Platform == PlatformOpenCodeGo {
		if err := NormalizeOpenCodeGoProtocolRulesCredentials(credentials); err != nil {
			return nil, ErrSupplierCredentialsInvalid.WithCause(err)
		}
	}
	if rawURL, exists := credentials["base_url"]; exists {
		if _, ok := rawURL.(string); !ok {
			return nil, ErrSupplierCredentialsInvalid
		}
	}
	if rawURLs, exists := credentials["api_base_urls"]; exists {
		if _, ok := rawURLs.(map[string]any); !ok {
			return nil, ErrSupplierCredentialsInvalid
		}
	}
	if err := validateSupplierCredentialConditions(s.cfg, SupplierAccountKind{Platform: account.Platform, Type: account.Type}, credentials); err != nil {
		return nil, ErrSupplierCredentialsInvalid.WithCause(err)
	}
	return credentials, nil
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
		if input.Status == "" {
			input.Status = StatusActive
		}
		if input.Status != StatusActive && input.Status != StatusDisabled {
			return ErrSupplierAccountInputInvalid
		}
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
		if input.Status != "" {
			return ErrSupplierAccountInputInvalid
		}
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
