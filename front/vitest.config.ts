import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    // Exclude Playwright e2e tests from Vitest
    exclude: ['**/node_modules/**', '**/tests/e2e/**'],
    include: ['app/**/*.test.ts', 'app/**/*.test.tsx'],
    environment: 'happy-dom',
    globals: true,
  },
})