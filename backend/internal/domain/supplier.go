package domain

const (
	SupplierStatusActive   = "active"
	SupplierStatusDisabled = "disabled"

	AccountReviewStatusPending  = "pending"
	AccountReviewStatusApproved = "approved"
	AccountReviewStatusRejected = "rejected"
)

// SupplierAccountKind identifies one platform and account type a supplier may submit.
type SupplierAccountKind struct {
	Platform string `json:"platform"`
	Type     string `json:"type"`
}
