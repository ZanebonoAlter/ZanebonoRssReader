import { describe, expect, it } from 'vitest'

import { resolveApiBaseUrlFromConfig, resolveApiOriginFromConfig } from './api'

describe('api runtime config', () => {
  it('prefers the internal API base on the server', () => {
    const config = {
      apiInternalBase: 'http://backend:5000/api',
      public: {
        apiBase: 'http://localhost:5000/api',
        apiOrigin: 'http://localhost:5000',
      },
    }

    expect(resolveApiBaseUrlFromConfig(config, true)).toBe('http://backend:5000/api')
  })

  it('uses the public API base on the client', () => {
    const config = {
      apiInternalBase: 'http://backend:5000/api',
      public: {
        apiBase: 'http://localhost:5000/api',
        apiOrigin: 'http://localhost:5000',
      },
    }

    expect(resolveApiBaseUrlFromConfig(config, false)).toBe('http://localhost:5000/api')
  })

  it('derives the origin from apiBase when apiOrigin is missing', () => {
    expect(resolveApiOriginFromConfig({
      public: {
        apiBase: 'http://localhost:5000/api',
      },
    })).toBe('http://localhost:5000')
  })
})
