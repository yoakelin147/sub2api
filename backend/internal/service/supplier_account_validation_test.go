package service

import (
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestValidateSupplierAccountCredentials(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		typeName string
		creds    map[string]any
		wantErr  bool
	}{
		{name: "openai api key", platform: PlatformOpenAI, typeName: AccountTypeAPIKey, creds: map[string]any{"api_key": " secret ", "base_url": "https://api.openai.com/"}},
		{name: "oauth token bundle with web login", platform: PlatformOpenAI, typeName: AccountTypeOAuth, creds: map[string]any{"access_token": "token", "refresh_token": "refresh", "email": "user@example.com", "password": "web-password"}},
		{name: "bedrock sigv4", platform: PlatformAnthropic, typeName: AccountTypeBedrock, creds: map[string]any{"auth_mode": "sigv4", "aws_region": "us-east-1", "aws_access_key_id": "id", "aws_secret_access_key": "secret"}},
		{name: "bedrock missing secret", platform: PlatformAnthropic, typeName: AccountTypeBedrock, creds: map[string]any{"auth_mode": "sigv4", "aws_region": "us-east-1", "aws_access_key_id": "id"}, wantErr: true},
		{name: "unknown field", platform: PlatformOpenAI, typeName: AccountTypeAPIKey, creds: map[string]any{"api_key": "secret", "priority": 1}, wantErr: true},
		{name: "oauth missing email", platform: PlatformGrok, typeName: AccountTypeOAuth, creds: map[string]any{"access_token": "token", "password": "secret"}, wantErr: true},
		{name: "api key password forbidden", platform: PlatformOpenAI, typeName: AccountTypeAPIKey, creds: map[string]any{"api_key": "token", "password": "secret"}, wantErr: true},
		{name: "private base url", platform: PlatformOpenAI, typeName: AccountTypeUpstream, creds: map[string]any{"api_key": "secret", "base_url": "https://127.0.0.1/v1"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateSupplierAccountCredentials(&config.Config{}, tt.platform, tt.typeName, tt.creds)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

func TestValidateSupplierAccountCredentialsValidatesServiceAccountJSON(t *testing.T) {
	_, err := ValidateSupplierAccountCredentials(&config.Config{}, PlatformGemini, AccountTypeServiceAccount, map[string]any{
		"service_account_json": `{"project_id":"p","client_email":"svc@example.com"}`,
		"location":             "global",
	})
	require.Error(t, err)
}

func TestValidateSupplierAccountCredentialsAcceptsValidServiceAccountJSON(t *testing.T) {
	raw, err := json.Marshal(map[string]string{
		"project_id": "project", "client_email": "svc@example.com",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("test")})),
	})
	require.NoError(t, err)

	credentials, err := ValidateSupplierAccountCredentials(&config.Config{}, PlatformGemini, AccountTypeServiceAccount, map[string]any{
		"service_account_json": string(raw), "project_id": "project", "client_email": "svc@example.com", "location": "us-central1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, credentials)
}

func TestValidateSupplierAccountCredentialsRequiresAdaptiveURLs(t *testing.T) {
	_, err := ValidateSupplierAccountCredentials(&config.Config{}, PlatformKimi, AccountTypeAPIKey, map[string]any{
		"api_key": "secret", "account_mode": AccountModePayG, "api_protocol": APIProtocolAdaptive,
	})
	require.Error(t, err)
}
