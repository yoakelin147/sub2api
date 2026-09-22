package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	suppliers *service.SupplierService
	tokens    *service.SupplierTokenService
	admin     service.AdminService
}

func NewSupplierHandler(suppliers *service.SupplierService, tokens *service.SupplierTokenService, adminService service.AdminService) *SupplierHandler {
	return &SupplierHandler{suppliers: suppliers, tokens: tokens, admin: adminService}
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
	response.Success(c, user)
}

func adminSupplierID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "Invalid supplier ID")
		return 0, false
	}
	return id, true
}
