package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const supplierTokenPrefix = "supplier_"

var (
	ErrSupplierTokenInvalid = infraerrors.Unauthorized(
		"INVALID_SUPPLIER_TOKEN",
		"invalid supplier token",
	)
	ErrSupplierDisabled = infraerrors.Forbidden("SUPPLIER_DISABLED", "supplier is disabled")
)

type SupplierTokenService struct {
	repo SupplierRepository
}

func NewSupplierTokenService(repo SupplierRepository) *SupplierTokenService {
	return &SupplierTokenService{repo: repo}
}

func (s *SupplierTokenService) Regenerate(ctx context.Context, supplierID int64) (string, error) {
	supplier, err := s.repo.GetByID(ctx, supplierID)
	if err != nil {
		return "", err
	}
	selectorBytes := make([]byte, 12)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(selectorBytes); err != nil {
		return "", err
	}
	if _, err := rand.Read(secretBytes); err != nil {
		return "", err
	}
	selector := hex.EncodeToString(selectorBytes)
	secret := hex.EncodeToString(secretBytes)
	token := supplierTokenPrefix + selector + "_" + secret
	digest := sha256.Sum256([]byte(secret))
	hash := hex.EncodeToString(digest[:])
	prefix := token[:min(len(token), 18)]
	now := time.Now().UTC()
	supplier.TokenSelector = &selector
	supplier.TokenHash = &hash
	supplier.TokenPrefix = &prefix
	supplier.TokenCreatedAt = &now
	supplier.TokenLastUsedAt = nil
	if err := s.repo.Update(ctx, supplier); err != nil {
		return "", err
	}
	return token, nil
}

func (s *SupplierTokenService) Authenticate(ctx context.Context, token string) (*Supplier, error) {
	selector, secret, ok := parseSupplierToken(token)
	if !ok {
		return nil, ErrSupplierTokenInvalid
	}
	supplier, err := s.repo.GetByTokenSelector(ctx, selector)
	if err != nil || supplier.TokenHash == nil {
		return nil, ErrSupplierTokenInvalid
	}
	digest := sha256.Sum256([]byte(secret))
	expected, err := hex.DecodeString(*supplier.TokenHash)
	if err != nil || len(expected) != len(digest) || subtle.ConstantTimeCompare(expected, digest[:]) != 1 {
		return nil, ErrSupplierTokenInvalid
	}
	if supplier.Status != domain.SupplierStatusActive {
		return nil, ErrSupplierDisabled
	}
	now := time.Now().UTC()
	if supplier.TokenLastUsedAt == nil || now.Sub(*supplier.TokenLastUsedAt) >= 5*time.Minute {
		_ = s.repo.UpdateTokenLastUsed(ctx, supplier.ID, now)
	}
	return supplier, nil
}

func (s *SupplierTokenService) Revoke(ctx context.Context, supplierID int64) error {
	supplier, err := s.repo.GetByID(ctx, supplierID)
	if err != nil {
		return err
	}
	supplier.TokenSelector = nil
	supplier.TokenHash = nil
	supplier.TokenPrefix = nil
	supplier.TokenCreatedAt = nil
	supplier.TokenLastUsedAt = nil
	return s.repo.Update(ctx, supplier)
}

func parseSupplierToken(token string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(token), "_")
	if len(parts) != 3 || parts[0] != "supplier" || len(parts[1]) != 24 || len(parts[2]) != 64 {
		return "", "", false
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", "", false
	}
	if _, err := hex.DecodeString(parts[2]); err != nil {
		return "", "", false
	}
	return parts[1], parts[2], true
}
