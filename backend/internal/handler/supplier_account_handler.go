package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const maxSupplierAccountRequestBytes = 1 << 20
const maxSupplierAccountBatchRequestBytes = 2 << 20

var errSupplierRequestTooLarge = errors.New("supplier request body is too large")

type supplierAccountRequest struct {
	ExternalID         string         `json:"external_id"`
	Name               string         `json:"name"`
	Notes              *string        `json:"notes"`
	Platform           string         `json:"platform"`
	Type               string         `json:"type"`
	Credentials        map[string]any `json:"credentials"`
	Extra              map[string]any `json:"extra"`
	ExpiresAt          *int64         `json:"expires_at"`
	ProxyID            *int64         `json:"proxy_id"`
	Concurrency        *int           `json:"concurrency"`
	Priority           *int           `json:"priority"`
	LoadFactor         *int           `json:"load_factor"`
	AutoPauseOnExpired *bool          `json:"auto_pause_on_expired"`
	GroupIDs           []int64        `json:"group_ids"`
}

type supplierAccountUpdateRequest struct {
	ExternalID         *string         `json:"external_id"`
	Name               *string         `json:"name"`
	Notes              *string         `json:"notes"`
	Credentials        *map[string]any `json:"credentials"`
	Extra              *map[string]any `json:"extra"`
	ExpiresAt          *int64          `json:"expires_at"`
	Status             *string         `json:"status"`
	ProxyID            *int64          `json:"proxy_id"`
	Concurrency        *int            `json:"concurrency"`
	Priority           *int            `json:"priority"`
	LoadFactor         *int            `json:"load_factor"`
	AutoPauseOnExpired *bool           `json:"auto_pause_on_expired"`
	GroupIDs           []int64         `json:"group_ids"`
}

type supplierAccountBatchRequest struct {
	Accounts []supplierAccountRequest `json:"accounts"`
}

type supplierAccountTestRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Mode   string `json:"mode"`
}

func (h *SupplierHandler) ListProxies(c *gin.Context) {
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return
	}
	options, err := h.accounts.AvailableProxies(c.Request.Context(), supplierID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, options)
}

func (h *SupplierHandler) ListAccounts(c *gin.Context) {
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.accounts.List(c.Request.Context(), supplierID, pagination.PaginationParams{
		Page: page, PageSize: pageSize, SortBy: c.Query("sort_by"), SortOrder: c.Query("sort_order"),
	}, service.SupplierAccountFilters{
		Platform: c.Query("platform"), Type: c.Query("type"), Status: c.Query("status"),
		ReviewStatus: c.Query("review_status"), Search: c.Query("search"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, supplierAccountResponse(&items[i]))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *SupplierHandler) GetAccount(c *gin.Context) {
	supplierID, accountID, ok := supplierAndAccountIDs(c)
	if !ok {
		return
	}
	account, err := h.accounts.GetByID(c.Request.Context(), supplierID, accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, supplierAccountResponse(account))
}

func (h *SupplierHandler) CreateAccount(c *gin.Context) {
	var request supplierAccountRequest
	if err := decodeStrictSupplierJSON(c, &request); err != nil {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	executeSupplierIdempotentJSON(c, "supplier.accounts.create", request, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		supplierID, ok := middleware.GetSupplierIDFromContext(c)
		if !ok {
			return nil, service.ErrSupplierTokenInvalid
		}
		account, err := h.accounts.Create(ctx, supplierID, service.CreateSupplierAccountInput{
			ExternalID: request.ExternalID, Name: request.Name, Notes: request.Notes,
			Platform: request.Platform, Type: request.Type, Credentials: request.Credentials, Extra: request.Extra,
			ExpiresAt: unixTime(request.ExpiresAt),
			ProxyID:   request.ProxyID, Concurrency: request.Concurrency, Priority: request.Priority,
			LoadFactor: request.LoadFactor, AutoPauseOnExpired: request.AutoPauseOnExpired, GroupIDs: request.GroupIDs,
		})
		if err != nil {
			return nil, err
		}
		middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID, "account_id": account.ID})
		return supplierAccountResponse(account), nil
	})
}

func (h *SupplierHandler) BatchCreateAccounts(c *gin.Context) {
	var request supplierAccountBatchRequest
	if err := decodeStrictSupplierJSONLimit(c, &request, maxSupplierAccountBatchRequestBytes); err != nil {
		if errors.Is(err, errSupplierRequestTooLarge) {
			response.ErrorFrom(c, service.ErrSupplierAccountBatchTooLarge)
			return
		}
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	inputs := make([]service.CreateSupplierAccountInput, 0, len(request.Accounts))
	for _, item := range request.Accounts {
		inputs = append(inputs, service.CreateSupplierAccountInput{
			ExternalID: item.ExternalID, Name: item.Name, Notes: item.Notes,
			Platform: item.Platform, Type: item.Type, Credentials: item.Credentials, Extra: item.Extra,
			ExpiresAt: unixTime(item.ExpiresAt),
			ProxyID:   item.ProxyID, Concurrency: item.Concurrency, Priority: item.Priority,
			LoadFactor: item.LoadFactor, AutoPauseOnExpired: item.AutoPauseOnExpired, GroupIDs: item.GroupIDs,
		})
	}
	executeSupplierIdempotentJSON(c, "supplier.accounts.batch_create", request, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		supplierID, ok := middleware.GetSupplierIDFromContext(c)
		if !ok {
			return nil, service.ErrSupplierTokenInvalid
		}
		result, err := h.accounts.BatchCreate(ctx, supplierID, inputs)
		if err != nil {
			return nil, err
		}
		accountIDs := make([]string, 0, result.Succeeded)
		for _, item := range result.Results {
			if item.Success {
				accountIDs = append(accountIDs, strconv.FormatInt(item.AccountID, 10))
			}
		}
		middleware.SetAuditExtra(c, map[string]any{
			"supplier_id": supplierID, "account_ids": strings.Join(accountIDs, ","), "requested_count": result.Total,
		})
		return result, nil
	})
}

func (h *SupplierHandler) UpdateAccount(c *gin.Context) {
	supplierID, accountID, ok := supplierAndAccountIDs(c)
	if !ok {
		return
	}
	var request supplierAccountUpdateRequest
	if err := decodeStrictSupplierJSON(c, &request); err != nil {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	account, err := h.accounts.Update(c.Request.Context(), supplierID, accountID, service.UpdateSupplierAccountInput{
		ExternalID: request.ExternalID, Name: request.Name, Notes: request.Notes,
		Credentials: request.Credentials, Extra: request.Extra, ExpiresAt: unixTime(request.ExpiresAt), Status: request.Status,
		ProxyID: request.ProxyID, Concurrency: request.Concurrency, Priority: request.Priority,
		LoadFactor: request.LoadFactor, AutoPauseOnExpired: request.AutoPauseOnExpired, GroupIDs: request.GroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID, "account_id": account.ID})
	response.Success(c, supplierAccountResponse(account))
}

func (h *SupplierHandler) DeleteAccount(c *gin.Context) {
	supplierID, accountID, ok := supplierAndAccountIDs(c)
	if !ok {
		return
	}
	if err := h.accounts.Delete(c.Request.Context(), supplierID, accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID, "account_id": accountID})
	response.Success(c, gin.H{"message": "Supplier account deleted"})
}

func (h *SupplierHandler) TestAccount(c *gin.Context) {
	supplierID, accountID, ok := supplierAndAccountIDs(c)
	if !ok {
		return
	}
	var request supplierAccountTestRequest
	if err := decodeStrictSupplierJSON(c, &request); err != nil {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	account, err := h.accounts.GetByID(c.Request.Context(), supplierID, accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID, "account_id": accountID})
	_ = h.tester.TestAccount(c, account, request.Model, request.Prompt, request.Mode)
}

func supplierAndAccountIDs(c *gin.Context) (int64, int64, bool) {
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return 0, 0, false
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return 0, 0, false
	}
	return supplierID, accountID, true
}

func decodeStrictSupplierJSON(c *gin.Context, target any) error {
	return decodeStrictSupplierJSONLimit(c, target, maxSupplierAccountRequestBytes)
}

func decodeStrictSupplierJSONLimit(c *gin.Context, target any, limit int64) error {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, limit+1))
	if err != nil || int64(len(raw)) > limit {
		if err == nil && int64(len(raw)) > limit {
			return errSupplierRequestTooLarge
		}
		return service.ErrSupplierAccountInputInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return service.ErrSupplierAccountInputInvalid
	}
	return nil
}

func unixTime(value *int64) *time.Time {
	if value == nil {
		return nil
	}
	t := time.Unix(*value, 0).UTC()
	return &t
}

func supplierAccountResponse(account *service.Account) gin.H {
	_, credentialStatus := dto.RedactCredentials(account.Credentials)
	visibleExtra := make(map[string]any)
	for _, key := range []string{service.AccountExtraUpstreamRequestIDHeader, "openai_compact_mode", "web_search_emulation"} {
		if value, ok := account.Extra[key].(string); ok {
			visibleExtra[key] = value
		}
	}
	return gin.H{
		"id": account.ID, "external_id": account.SupplierExternalID, "name": account.Name,
		"notes": account.Notes, "platform": account.Platform, "type": account.Type,
		"extra":             visibleExtra,
		"credential_status": credentialStatus, "has_credentials": len(credentialStatus) > 0,
		"status": account.Status, "schedulable": account.Schedulable,
		"proxy_id": account.ProxyID, "concurrency": account.Concurrency,
		"group_ids": account.GroupIDs,
		"priority":  account.Priority, "load_factor": account.LoadFactor,
		"auto_pause_on_expired": account.AutoPauseOnExpired,
		"review_status":         account.ReviewStatus, "review_note": account.ReviewNote,
		"expires_at": account.ExpiresAt, "last_used_at": account.LastUsedAt,
		"created_at": account.CreatedAt, "updated_at": account.UpdatedAt,
	}
}
