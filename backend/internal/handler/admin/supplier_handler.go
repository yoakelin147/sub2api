package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	suppliers *service.SupplierService
	tokens    *service.SupplierTokenService
	admin     service.AdminService
	accounts  *service.SupplierAccountService
}

func NewSupplierHandler(suppliers *service.SupplierService, tokens *service.SupplierTokenService, adminService service.AdminService, accounts *service.SupplierAccountService) *SupplierHandler {
	return &SupplierHandler{suppliers: suppliers, tokens: tokens, admin: adminService, accounts: accounts}
}

type supplierWriteRequest struct {
	Code                string                       `json:"code" binding:"required"`
	Name                string                       `json:"name" binding:"required"`
	Status              string                       `json:"status" binding:"omitempty,oneof=active disabled"`
	Notes               *string                      `json:"notes"`
	AllowedAccountKinds []domain.SupplierAccountKind `json:"allowed_account_kinds"`
}

type supplierMemberRequest struct {
	UserID      *int64 `json:"user_id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	Concurrency int    `json:"concurrency"`
}

type supplierAccountReviewRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
	GroupIDs   []int64 `json:"group_ids"`
	Note       *string `json:"note"`
}

func (h *SupplierHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.suppliers.List(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SupplierListFilters{
		Status: c.Query("status"), Search: c.Query("search"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, adminSupplierResponse(&items[i]))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *SupplierHandler) Get(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	supplier, err := h.suppliers.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, adminSupplierResponse(supplier))
}

func (h *SupplierHandler) Create(c *gin.Context) {
	var request supplierWriteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	supplier, err := h.suppliers.Create(c.Request.Context(), service.CreateSupplierInput{
		Code: request.Code, Name: request.Name, Status: request.Status, Notes: request.Notes,
		AllowedAccountKinds: request.AllowedAccountKinds,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplier.ID})
	response.Success(c, adminSupplierResponse(supplier))
}

func (h *SupplierHandler) Update(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	var request supplierWriteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	name, kinds := request.Name, request.AllowedAccountKinds
	var status *string
	if request.Status != "" {
		status = &request.Status
	}
	supplier, err := h.suppliers.Update(c.Request.Context(), id, service.UpdateSupplierInput{
		Name: &name, Status: status, Notes: request.Notes, AllowedAccountKinds: &kinds,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": id})
	response.Success(c, adminSupplierResponse(supplier))
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	if err := h.suppliers.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": id})
	response.Success(c, gin.H{"message": "Supplier deleted"})
}

func (h *SupplierHandler) AccessTokenStatus(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	supplier, err := h.suppliers.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"exists": supplier.TokenHash != nil, "masked_key": supplier.TokenPrefix, "created_at": supplier.TokenCreatedAt, "last_used_at": supplier.TokenLastUsedAt})
}

func (h *SupplierHandler) RegenerateAccessToken(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	token, err := h.tokens.Regenerate(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": id})
	response.Success(c, gin.H{"key": token})
}

func (h *SupplierHandler) RevokeAccessToken(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	if err := h.tokens.Revoke(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": id})
	response.Success(c, gin.H{"message": "Supplier access token revoked"})
}

func (h *SupplierHandler) ListMembers(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	includeSubscriptions := false
	users, total, err := h.admin.ListUsers(c.Request.Context(), page, pageSize, service.UserListFilters{
		Role: service.RoleSupplier, SupplierID: &id, IncludeSubscriptions: &includeSubscriptions,
	}, "created_at", "desc")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminUser, 0, len(users))
	for i := range users {
		out = append(out, *dto.UserFromServiceAdmin(&users[i]))
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *SupplierHandler) AddMember(c *gin.Context) {
	id, ok := adminSupplierID(c)
	if !ok {
		return
	}
	var request supplierMemberRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if request.UserID != nil {
		user, err := h.admin.UpdateUser(c.Request.Context(), *request.UserID, &service.UpdateUserInput{
			Role: service.RoleSupplier, SupplierID: &id,
		})
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		middleware.SetAuditExtra(c, map[string]any{"supplier_id": id, "member_user_id": user.ID})
		response.Success(c, dto.UserFromServiceAdmin(user))
		return
	}
	if request.Email == "" || request.Password == "" {
		response.BadRequest(c, "email and password are required for a new member")
		return
	}
	user, err := h.admin.CreateUser(c.Request.Context(), &service.CreateUserInput{
		Email: request.Email, Password: request.Password, Username: request.Username,
		Role: service.RoleSupplier, SupplierID: &id, Concurrency: request.Concurrency,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": id, "member_user_id": user.ID})
	response.Success(c, dto.UserFromServiceAdmin(user))
}

func adminSupplierResponse(supplier *service.Supplier) gin.H {
	return gin.H{
		"id": supplier.ID, "code": supplier.Code, "name": supplier.Name,
		"status": supplier.Status, "notes": supplier.Notes,
		"allowed_account_kinds": supplier.AllowedAccountKinds,
		"access_token": gin.H{
			"exists": supplier.TokenHash != nil, "masked_key": supplier.TokenPrefix,
			"created_at": supplier.TokenCreatedAt, "last_used_at": supplier.TokenLastUsedAt,
		},
		"created_at": supplier.CreatedAt, "updated_at": supplier.UpdatedAt,
	}
}

func (h *SupplierHandler) RemoveMember(c *gin.Context) {
	supplierID, ok := adminSupplierID(c)
	if !ok {
		return
	}
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	user, err := h.admin.GetUser(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if user.Role != service.RoleSupplier || user.SupplierID == nil || *user.SupplierID != supplierID {
		response.ErrorFrom(c, service.ErrUserNotFound)
		return
	}
	user, err = h.admin.UpdateUser(c.Request.Context(), userID, &service.UpdateUserInput{
		Role: service.RoleUser, Status: service.StatusDisabled,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID, "member_user_id": userID})
	response.Success(c, user)
}

func (h *SupplierHandler) ListAccounts(c *gin.Context) {
	supplierID, ok := adminSupplierID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.accounts.ListForAdmin(c.Request.Context(), supplierID, pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SupplierAccountFilters{
		Platform: c.Query("platform"), Type: c.Query("type"), Status: c.Query("status"),
		ReviewStatus: c.Query("review_status"), Search: c.Query("search"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.Account, 0, len(items))
	for i := range items {
		out = append(out, *dto.AccountFromServiceShallow(&items[i]))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *SupplierHandler) ApproveAccounts(c *gin.Context) {
	h.reviewAccounts(c, service.SupplierAccountReviewApprove)
}
func (h *SupplierHandler) RejectAccounts(c *gin.Context) {
	h.reviewAccounts(c, service.SupplierAccountReviewReject)
}
func (h *SupplierHandler) PauseAccounts(c *gin.Context) {
	h.reviewAccounts(c, service.SupplierAccountReviewPause)
}

func (h *SupplierHandler) reviewAccounts(c *gin.Context, action string) {
	supplierID, ok := adminSupplierID(c)
	if !ok {
		return
	}
	var request supplierAccountReviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Admin not authenticated")
		return
	}
	err := h.accounts.Review(c.Request.Context(), supplierID, service.SupplierAccountReviewInput{
		AccountIDs: request.AccountIDs, Action: action, GroupIDs: request.GroupIDs,
		ReviewerID: subject.UserID, Note: request.Note,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	accountIDs := make([]string, 0, len(request.AccountIDs))
	for _, accountID := range request.AccountIDs {
		accountIDs = append(accountIDs, strconv.FormatInt(accountID, 10))
	}
	middleware.SetAuditExtra(c, map[string]any{
		"supplier_id": supplierID, "account_ids": strings.Join(accountIDs, ","), "requested_count": len(request.AccountIDs),
	})
	response.Success(c, gin.H{"updated": len(request.AccountIDs), "action": action})
}

func adminSupplierID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "Invalid supplier ID")
		return 0, false
	}
	return id, true
}
