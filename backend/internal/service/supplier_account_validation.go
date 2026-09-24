package service

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const maxSupplierCredentialBytes = 1 << 20

var ErrSupplierCredentialsInvalid = infraerrors.New(
	http.StatusUnprocessableEntity,
	"INVALID_CREDENTIALS",
	"supplier account credentials are invalid",
)

type supplierCredentialSpec struct {
	required map[string]struct{}
	allowed  map[string]struct{}
}

func credentialSpec(required []string, optional ...string) supplierCredentialSpec {
	allowed := make(map[string]struct{}, len(required)+len(optional))
	requiredSet := make(map[string]struct{}, len(required))
	for _, key := range required {
		requiredSet[key] = struct{}{}
		allowed[key] = struct{}{}
	}
	for _, key := range optional {
		allowed[key] = struct{}{}
	}
	return supplierCredentialSpec{required: requiredSet, allowed: allowed}
}

var supplierCredentialSpecs = map[SupplierAccountKind]supplierCredentialSpec{
	{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}:            credentialSpec([]string{"api_key"}, "base_url"),
	{Platform: PlatformOpenAI, Type: AccountTypeUpstream}:          credentialSpec([]string{"api_key", "base_url"}),
	{Platform: PlatformOpenAI, Type: AccountTypeOAuth}:             credentialSpec([]string{"access_token", "email", "password"}, "refresh_token", "id_token", "expires_at", "chatgpt_account_id", "chatgpt_user_id", "organization_id", "plan_type", "subscription_expires_at", "client_id"),
	{Platform: PlatformOpenAI, Type: AccountTypeSetupToken}:        credentialSpec([]string{"access_token", "email", "password"}, "refresh_token", "expires_at", "chatgpt_account_id"),
	{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}:         credentialSpec([]string{"api_key"}, "base_url", "auth_scheme"),
	{Platform: PlatformAnthropic, Type: AccountTypeUpstream}:       credentialSpec([]string{"api_key", "base_url"}, "auth_scheme"),
	{Platform: PlatformAnthropic, Type: AccountTypeOAuth}:          credentialSpec([]string{"access_token", "email", "password"}, "refresh_token", "expires_at", "token_type", "scope"),
	{Platform: PlatformAnthropic, Type: AccountTypeSetupToken}:     credentialSpec([]string{"access_token", "email", "password"}, "refresh_token", "expires_at", "token_type", "scope"),
	{Platform: PlatformAnthropic, Type: AccountTypeBedrock}:        credentialSpec([]string{"auth_mode", "aws_region"}, "aws_access_key_id", "aws_secret_access_key", "aws_session_token", "aws_force_global", "api_key"),
	{Platform: PlatformAnthropic, Type: AccountTypeServiceAccount}: credentialSpec([]string{"service_account_json", "location"}, "project_id", "client_email", "tier_id"),
	{Platform: PlatformGemini, Type: AccountTypeAPIKey}:            credentialSpec([]string{"api_key"}, "base_url", "tier_id"),
	{Platform: PlatformGemini, Type: AccountTypeOAuth}:             credentialSpec([]string{"access_token", "email", "password"}, "refresh_token", "token_type", "expires_at", "scope", "project_id", "oauth_type", "tier_id"),
	{Platform: PlatformGemini, Type: AccountTypeServiceAccount}:    credentialSpec([]string{"service_account_json", "location"}, "project_id", "client_email", "tier_id"),
	{Platform: PlatformAntigravity, Type: AccountTypeOAuth}:        credentialSpec([]string{"access_token", "email", "password"}, "refresh_token", "token_type", "expires_at", "project_id", "plan_type"),
	{Platform: PlatformAntigravity, Type: AccountTypeAPIKey}:       credentialSpec([]string{"api_key", "base_url"}),
	{Platform: PlatformGrok, Type: AccountTypeAPIKey}:              credentialSpec([]string{"api_key"}, "base_url"),
	{Platform: PlatformGrok, Type: AccountTypeOAuth}:               credentialSpec([]string{"access_token", "email", "password"}, "refresh_token", "id_token", "token_type", "expires_at", "client_id", "scope", "sub", "team_id", "subscription_tier", "entitlement_status", "base_url"),
	{Platform: PlatformKimi, Type: AccountTypeAPIKey}:              credentialSpec([]string{"api_key", "account_mode", "api_protocol"}, "base_url", "api_base_urls"),
	{Platform: PlatformZhipu, Type: AccountTypeAPIKey}:             credentialSpec([]string{"api_key", "account_mode", "api_protocol"}, "base_url", "api_base_urls", "zhipu_organization", "zhipu_project"),
	{Platform: PlatformDeepseek, Type: AccountTypeAPIKey}:          credentialSpec([]string{"api_key", "account_mode", "api_protocol"}, "base_url", "api_base_urls"),
	{Platform: PlatformMiniMax, Type: AccountTypeAPIKey}:           credentialSpec([]string{"api_key", "account_mode", "api_protocol"}, "base_url", "api_base_urls"),
	{Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey}:        credentialSpec([]string{"api_key", "account_mode", "api_protocol"}, "base_url", "api_base_urls", "protocol_rules"),
}

var supplierConnectionCredentialKeys = map[string]struct{}{
	"api_key": {}, "access_token": {}, "refresh_token": {}, "id_token": {}, "expires_at": {},
	"email": {}, "password": {}, "base_url": {}, "api_base_urls": {}, "auth_scheme": {},
	"auth_mode": {}, "aws_region": {}, "aws_access_key_id": {}, "aws_secret_access_key": {},
	"aws_session_token": {}, "service_account_json": {}, "location": {}, "project_id": {},
	"client_email": {}, "account_mode": {}, "api_protocol": {}, "chatgpt_account_id": {},
	"chatgpt_user_id": {}, "organization_id": {}, "client_id": {}, "token_type": {},
	"scope": {}, "oauth_type": {}, "aws_force_global": {}, "zhipu_organization": {},
	"zhipu_project": {},
}

func validateSupplierCredentialPermissions(platform, accountType string, credentials map[string]any, reviewRequired bool) error {
	if !reviewRequired {
		return nil
	}
	spec, ok := supplierCredentialSpecs[SupplierAccountKind{Platform: platform, Type: accountType}]
	if !ok {
		return ErrSupplierCredentialsInvalid
	}
	for key := range credentials {
		if _, allowed := spec.allowed[key]; !allowed {
			return ErrSupplierCredentialsInvalid
		}
		if _, allowed := supplierConnectionCredentialKeys[key]; !allowed {
			return ErrSupplierCredentialsInvalid
		}
	}
	return nil
}

func ValidateSupplierAccountCredentials(cfg *config.Config, platform, accountType string, credentials map[string]any) (map[string]any, error) {
	kind := SupplierAccountKind{Platform: strings.ToLower(strings.TrimSpace(platform)), Type: strings.ToLower(strings.TrimSpace(accountType))}
	spec, ok := supplierCredentialSpecs[kind]
	if !ok {
		return nil, ErrSupplierAccountKindInvalid
	}
	encoded, err := json.Marshal(credentials)
	if err != nil || len(encoded) > maxSupplierCredentialBytes {
		return nil, ErrSupplierCredentialsInvalid
	}
	normalized := make(map[string]any, len(credentials))
	for key, value := range credentials {
		if _, ok := spec.allowed[key]; !ok {
			if key != "model_mapping" && key != "compact_model_mapping" {
				return nil, ErrSupplierCredentialsInvalid
			}
			if err := validateSupplierModelMapping(value); err != nil {
				return nil, err
			}
		}
		if text, ok := value.(string); ok {
			if key != "password" {
				value = strings.TrimSpace(text)
			}
			if len(text) > maxSupplierCredentialBytes {
				return nil, ErrSupplierCredentialsInvalid
			}
		}
		normalized[key] = value
	}
	for key := range spec.required {
		if !nonEmptyCredentialValue(normalized[key]) {
			return nil, ErrSupplierCredentialsInvalid
		}
	}
	if kind.Type == AccountTypeOAuth || kind.Type == AccountTypeSetupToken {
		email, ok := normalized["email"].(string)
		address, err := mail.ParseAddress(email)
		if !ok || err != nil || address.Address != email {
			return nil, ErrSupplierCredentialsInvalid
		}
		if _, ok := normalized["password"].(string); !ok {
			return nil, ErrSupplierCredentialsInvalid
		}
	}
	if kind.Platform == PlatformOpenCodeGo {
		if err := NormalizeOpenCodeGoProtocolRulesCredentials(normalized); err != nil {
			return nil, ErrSupplierCredentialsInvalid.WithCause(err)
		}
	}
	if err := validateSupplierCredentialConditions(cfg, kind, normalized); err != nil {
		return nil, ErrSupplierCredentialsInvalid.WithCause(err)
	}
	return normalized, nil
}

func validateSupplierModelMapping(value any) error {
	mapping, ok := value.(map[string]any)
	if !ok || len(mapping) > 256 {
		return ErrSupplierCredentialsInvalid
	}
	for source, target := range mapping {
		model, ok := target.(string)
		if strings.TrimSpace(source) == "" || len(source) > 256 || !ok || strings.TrimSpace(model) == "" || len(model) > 256 {
			return ErrSupplierCredentialsInvalid
		}
	}
	return nil
}

func validateSupplierExtra(platform, accountType string, extra map[string]any) error {
	encoded, err := json.Marshal(extra)
	if err != nil || len(encoded) > 1<<16 {
		return ErrSupplierAccountInputInvalid
	}
	for key, value := range extra {
		switch key {
		case AccountExtraUpstreamRequestIDHeader:
		case "openai_compact_mode":
			mode, ok := value.(string)
			if platform != PlatformOpenAI || !ok || mode != OpenAICompactModeAuto && mode != OpenAICompactModeForceOn && mode != OpenAICompactModeForceOff {
				return ErrSupplierAccountInputInvalid
			}
		case featureKeyWebSearchEmulation:
			mode, ok := value.(string)
			if platform != PlatformAnthropic || accountType != AccountTypeAPIKey || !ok || mode != WebSearchModeDefault && mode != WebSearchModeEnabled && mode != WebSearchModeDisabled {
				return ErrSupplierAccountInputInvalid
			}
		default:
			return ErrSupplierAccountInputInvalid
		}
	}
	if err := ValidateUpstreamRequestIDHeaderExtra(extra); err != nil {
		return ErrSupplierAccountInputInvalid.WithCause(err)
	}
	return nil
}

func nonEmptyCredentialValue(value any) bool {
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value) != ""
	case map[string]any:
		return len(value) > 0
	default:
		return value != nil
	}
}

func validateSupplierCredentialConditions(cfg *config.Config, kind SupplierAccountKind, credentials map[string]any) error {
	if baseURL, ok := credentials["base_url"].(string); ok && baseURL != "" {
		normalized, err := validateSupplierBaseURL(cfg, baseURL)
		if err != nil {
			return err
		}
		credentials["base_url"] = normalized
	}
	if urls, ok := credentials["api_base_urls"].(map[string]any); ok {
		for protocol, raw := range urls {
			value, ok := raw.(string)
			if !ok {
				return ErrSupplierCredentialsInvalid
			}
			normalized, err := validateSupplierBaseURL(cfg, value)
			if err != nil {
				return err
			}
			urls[protocol] = normalized
		}
	}
	if kind.Type == AccountTypeBedrock {
		mode, _ := credentials["auth_mode"].(string)
		switch mode {
		case "sigv4":
			if !nonEmptyCredentialValue(credentials["aws_access_key_id"]) || !nonEmptyCredentialValue(credentials["aws_secret_access_key"]) {
				return ErrSupplierCredentialsInvalid
			}
		case "api_key":
			if !nonEmptyCredentialValue(credentials["api_key"]) {
				return ErrSupplierCredentialsInvalid
			}
		default:
			return ErrSupplierCredentialsInvalid
		}
	}
	if kind.Type == AccountTypeServiceAccount {
		key, err := parseVertexServiceAccountKey(&Account{Credentials: credentials})
		if err != nil {
			return err
		}
		block, _ := pem.Decode([]byte(key.PrivateKey))
		if block == nil || !strings.Contains(block.Type, "PRIVATE KEY") {
			return ErrSupplierCredentialsInvalid
		}
		if projectID, ok := credentials["project_id"].(string); ok && projectID != "" && projectID != key.ProjectID {
			return ErrSupplierCredentialsInvalid
		}
		if email, ok := credentials["client_email"].(string); ok && email != "" && email != key.ClientEmail {
			return ErrSupplierCredentialsInvalid
		}
		location, _ := credentials["location"].(string)
		if !vertexLocationPattern.MatchString(location) {
			return ErrSupplierCredentialsInvalid
		}
	}
	if kind.Platform == PlatformAnthropic {
		if scheme, ok := credentials["auth_scheme"].(string); ok && scheme != "" && scheme != "x_api_key" && scheme != "authorization_bearer" {
			return ErrSupplierCredentialsInvalid
		}
	}
	if isSupplierCNPlatform(kind.Platform) || kind.Platform == PlatformOpenCodeGo {
		mode, _ := credentials["account_mode"].(string)
		if kind.Platform == PlatformOpenCodeGo {
			if mode != AccountModeZen && mode != AccountModeGo {
				return ErrSupplierCredentialsInvalid
			}
		} else if mode != AccountModePayG && mode != AccountModeCoding {
			return ErrSupplierCredentialsInvalid
		}
		protocol, _ := credentials["api_protocol"].(string)
		switch protocol {
		case APIProtocolChatCompletions, APIProtocolAnthropic, APIProtocolResponses, APIProtocolAdaptive:
		default:
			return ErrSupplierCredentialsInvalid
		}
		if protocol == APIProtocolAdaptive {
			urls, ok := credentials["api_base_urls"].(map[string]any)
			if !ok || len(urls) == 0 {
				return ErrSupplierCredentialsInvalid
			}
		}
	}
	return nil
}

func isSupplierCNPlatform(platform string) bool {
	switch platform {
	case PlatformKimi, PlatformZhipu, PlatformDeepseek, PlatformMiniMax:
		return true
	default:
		return false
	}
}

func validateSupplierBaseURL(cfg *config.Config, raw string) (string, error) {
	allowedHosts := []string{"api.openai.com", "api.anthropic.com", "generativelanguage.googleapis.com", "api.x.ai", "api.moonshot.cn", "open.bigmodel.cn", "api.deepseek.com", "api.minimaxi.com", "opencode.ai"}
	if cfg != nil {
		allowedHosts = append(allowedHosts, cfg.Security.URLAllowlist.UpstreamHosts...)
	}
	return urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts: allowedHosts, RequireAllowlist: true, AllowPrivate: false,
	})
}
