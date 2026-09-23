import type { SupplierAccountKind } from '@/api/supplier'

export const SUPPLIER_ACCOUNT_KINDS: SupplierAccountKind[] = [
  { platform: 'openai', type: 'apikey' },
  { platform: 'openai', type: 'upstream' },
  { platform: 'openai', type: 'oauth' },
  { platform: 'openai', type: 'setup-token' },
  { platform: 'anthropic', type: 'apikey' },
  { platform: 'anthropic', type: 'upstream' },
  { platform: 'anthropic', type: 'oauth' },
  { platform: 'anthropic', type: 'setup-token' },
  { platform: 'anthropic', type: 'bedrock' },
  { platform: 'anthropic', type: 'service_account' },
  { platform: 'gemini', type: 'apikey' },
  { platform: 'gemini', type: 'oauth' },
  { platform: 'gemini', type: 'service_account' },
  { platform: 'antigravity', type: 'oauth' },
  { platform: 'antigravity', type: 'apikey' },
  { platform: 'grok', type: 'apikey' },
  { platform: 'grok', type: 'oauth' },
  { platform: 'kimi', type: 'apikey' },
  { platform: 'zhipu', type: 'apikey' },
  { platform: 'deepseek', type: 'apikey' },
  { platform: 'minimax', type: 'apikey' },
  { platform: 'opencode_go', type: 'apikey' },
]

// Shared by the supplier form and API examples; the backend remains authoritative.
export function supplierCredentialTemplate(kind: SupplierAccountKind | undefined): Record<string, unknown> {
  if (!kind) return {}
  if (kind.type === 'oauth' || kind.type === 'setup-token') return { email: '', password: '', access_token: '', refresh_token: '' }
  if (kind.type === 'bedrock') return { auth_mode: 'api_key', aws_region: 'us-east-1', api_key: '' }
  if (kind.type === 'service_account') {
    return { service_account_json: '{"project_id":"","client_email":"","private_key":""}', location: 'us-central1' }
  }
  if (['kimi', 'zhipu', 'deepseek', 'minimax', 'opencode_go'].includes(kind.platform)) {
    return { api_key: '', account_mode: kind.platform === 'opencode_go' ? 'go' : 'payg', api_protocol: 'chat_completions' }
  }
  const template: Record<string, unknown> = { api_key: '' }
  if (kind.type === 'upstream' || kind.platform === 'antigravity') template.base_url = 'https://'
  return template
}

export function supplierRequiredCredentials(kind: SupplierAccountKind, credentials: Record<string, unknown> = {}): string[] {
  if (kind.type === 'oauth' || kind.type === 'setup-token') return ['email', 'password', 'access_token']
  if (kind.type === 'bedrock') {
    return ['auth_mode', 'aws_region', ...(credentials.auth_mode === 'sigv4' ? ['aws_access_key_id', 'aws_secret_access_key'] : ['api_key'])]
  }
  if (kind.type === 'service_account') return ['service_account_json', 'location']
  return Object.keys(supplierCredentialTemplate(kind))
}
