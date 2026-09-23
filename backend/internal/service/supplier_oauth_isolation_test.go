package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

func TestSupplierOAuthSessionsRejectCrossTenantAndAdminExchange(t *testing.T) {
	ctx := context.Background()
	claude := &OAuthService{sessionStore: oauth.NewSessionStore()}
	defer claude.sessionStore.Stop()
	claude.sessionStore.Set("claude-session", &oauth.OAuthSession{SupplierID: 42, Scope: oauth.ScopeInference, CreatedAt: time.Now()})
	for _, supplierID := range []int64{0, 43} {
		_, err := claude.ExchangeSupplierCode(ctx, supplierID, true, &ExchangeCodeInput{SessionID: "claude-session", Code: "code"})
		require.ErrorContains(t, err, "session not found")
	}
	_, err := claude.ExchangeSupplierCode(ctx, 42, false, &ExchangeCodeInput{SessionID: "claude-session", Code: "code"})
	require.ErrorContains(t, err, "session not found")

	gemini := &GeminiOAuthService{sessionStore: geminicli.NewSessionStore()}
	defer gemini.sessionStore.Stop()
	gemini.sessionStore.Set("gemini-session", &geminicli.OAuthSession{SupplierID: 42, State: "expected", OAuthType: "code_assist", CreatedAt: time.Now()})
	for _, supplierID := range []int64{0, 43} {
		_, err = gemini.ExchangeSupplierCode(ctx, supplierID, &GeminiExchangeCodeInput{SessionID: "gemini-session", Code: "code", State: "expected"})
		require.ErrorContains(t, err, "session not found")
	}

	ag := &AntigravityOAuthService{sessionStore: antigravity.NewSessionStore()}
	defer ag.sessionStore.Stop()
	ag.sessionStore.Set("ag-session", &antigravity.OAuthSession{SupplierID: 42, State: "expected", CreatedAt: time.Now()})
	for _, supplierID := range []int64{0, 43} {
		_, err = ag.ExchangeSupplierCode(ctx, supplierID, &AntigravityExchangeCodeInput{SessionID: "ag-session", Code: "code", State: "expected"})
		require.ErrorContains(t, err, "session")
	}

	grok := &GrokOAuthService{sessionStore: xai.NewSessionStore()}
	defer grok.sessionStore.Stop()
	grok.sessionStore.Set("grok-session", &xai.OAuthSession{SupplierID: 42, State: "expected", CreatedAt: time.Now()})
	for _, supplierID := range []int64{0, 43} {
		_, err = grok.ExchangeSupplierCode(ctx, supplierID, &GrokExchangeCodeInput{SessionID: "grok-session", Code: "code", State: "expected"})
		require.ErrorContains(t, err, "session not found")
	}
}
