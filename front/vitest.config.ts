import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    // Exclude Playwright e2e tests from Vitest
    exclude: ['**/node_modules/**', '**/tests/e2e/**'],
    include: ['app/**/*.test.ts', 'app/**/*.test.tsx'],
  },
})