const DEFAULT_API_ORIGIN = 'http://localhost:5000'
const DEFAULT_API_BASE = `${DEFAULT_API_ORIGIN}/api`

export interface RuntimeApiConfig {
  apiInternalBase?: string
  public?: {
    apiOrigin?: string
    apiBase?: string
  }
}

function trimValue(value?: string): string {
  return value?.trim() || ''
}

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function deriveOriginFromBase(base: string): string {
  return stripTrailingSlash(base).replace(/\/api$/, '')
}

export function resolveApiBaseUrlFromConfig(config?: RuntimeApiConfig, isServer = false): string {
  const internalBase = trimValue(config?.apiInternalBase)
  if (isServer && internalBase) {
    return stripTrailingSlash(internalBase)
  }

  const publicBase = trimValue(config?.public?.apiBase)
  if (publicBase) {
    return stripTrailingSlash(publicBase)
  }

  const publicOrigin = trimValue(config?.public?.apiOrigin)
  if (publicOrigin) {
    return `${stripTrailingSlash(publicOrigin)}/api`
  }

  return DEFAULT_API_BASE
}

export function resolveApiOriginFromConfig(config?: RuntimeApiConfig): string {
  const publicOrigin = trimValue(config?.public?.apiOrigin)
  if (publicOrigin) {
    return stripTrailingSlash(publicOrigin)
  }

  const publicBase = trimValue(config?.public?.apiBase)
  if (publicBase) {
    return deriveOriginFromBase(publicBase)
  }

  return DEFAULT_API_ORIGIN
}

function getRuntimeApiConfig(): RuntimeApiConfig | undefined {
  try {
    return useRuntimeConfig() as RuntimeApiConfig
  } catch {
    return undefined
  }
}

export function getApiBaseUrl(): string {
  return resolveApiBaseUrlFromConfig(getRuntimeApiConfig(), import.meta.server)
}

export function getApiOrigin(): string {
  return resolveApiOriginFromConfig(getRuntimeApiConfig())
}
