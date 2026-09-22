package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const maxSupplierAccountRequestBytes = 1 << 20

type supplierAccountRequest struct {
	ExternalID  string         `json:"external_id"`
	Name        string         `json:"name"`
	Notes       *string        `json:"notes"`
	Platform    string         `json:"platform"`
	Type        string         `json:"type"`
	Credentials map[string]any `json:"credentials"`
	ExpiresAt   *int64         `json:"expires_at"`
}

type supplierAccountUpdateRequest struct {
	ExternalID  *string         `json:"external_id"`
	Name        *string         `json:"name"`
	Notes       *string         `json:"notes"`
	Credentials *map[string]any `json:"credentials"`
	ExpiresAt   *int64          `json:"expires_at"`
	Status      *string         `json:"status"`
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
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return
	}
	var request supplierAccountRequest
	if err := decodeStrictSupplierJSON(c, &request); err != nil {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	account, err := h.accounts.Create(c.Request.Context(), supplierID, service.CreateSupplierAccountInput{
		ExternalID: request.ExternalID, Name: request.Name, Notes: request.Notes,
		Platform: request.Platform, Type: request.Type, Credentials: request.Credentials,
		ExpiresAt: unixTime(request.ExpiresAt),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, supplierAccountResponse(account))
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
		Credentials: request.Credentials, ExpiresAt: unixTime(request.ExpiresAt), Status: request.Status,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
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
	response.Success(c, gin.H{"message": "Supplier account deleted"})
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
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, maxSupplierAccountRequestBytes+1))
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
	return gin.H{
		"id": account.ID, "external_id": account.SupplierExternalID, "name": account.Name,
		"notes": account.Notes, "platform": account.Platform, "type": account.Type,
		"credential_status": credentialStatus, "has_credentials": len(credentialStatus) > 0,
		"status": account.Status, "schedulable": account.Schedulable,
		"review_status": account.ReviewStatus, "review_note": account.ReviewNote,
		"expires_at": account.ExpiresAt, "last_used_at": account.LastUsedAt,
		"created_at": account.CreatedAt, "updated_at": account.UpdatedAt,
	}
}
