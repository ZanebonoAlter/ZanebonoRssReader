# Frontend Agent Guide

**Scope:** `front/` — Nuxt 4 + Vue 3 + TypeScript frontend

## Overview
RSS Reader frontend using Vue 3 Composition API with `<script setup>`, Pinia for state management, and Tailwind CSS v4.

## Structure
```
front/
├── app/
│   ├── api/           # HTTP client layer
│   ├── components/    # Shared UI components
│   ├── composables/   # Vue composables
│   ├── features/      # Domain-specific modules
│   ├── pages/         # Nuxt routes
│   ├── stores/        # Pinia stores
│   ├── types/         # Shared TypeScript types
│   └── utils/         # Utility functions
├── public/            # Static assets
└── nuxt.config.ts
```

## Where to Look
| Task | Location |
|------|----------|
| API calls | `app/api/` |
| Shared components | `app/components/` |
| Feature modules | `app/features/*/` |
| Global state | `app/stores/` |
| Types | `app/types/` |

## Conventions

### Vue Files
- Always use `<script setup lang="ts">`
- Components: PascalCase (e.g., `ArticleCard.vue`)
- Composables: camelCase with `use` prefix (e.g., `useApiStore`)
- Props interfaces named `Props`

### Imports
```typescript
// Order: Vue/Nuxt → third-party → internal → types
import { ref, computed } from 'vue'
import { useRoute } from 'nuxt/app'
import { useDebounceFn } from '@vueuse/core'
import { useApiStore } from '~/stores/api'
import type { Article } from '~/types/article'
```

### API Pattern
All HTTP calls go through `app/api/client.ts`:
```typescript
// Returns { success, data, error, message }
const result = await apiClient.getArticles()
if (!result.success) {
  // handle error
}
```

### Store Pattern
```typescript
// Primary data source pattern
export const useApiStore = defineStore('api', () => {
  const articles = ref<Article[]>([])
  // UI stores derive from this
  return { articles }
})
```

### Data Mapping
- Convert numeric IDs to strings at API boundary
- Map `snake_case` → `camelCase` in store/API code only
- Never map in templates

## Anti-Patterns
- DON'T put API calls directly in components
- DON'T use `any` types
- DON'T suppress TypeScript errors with `@ts-ignore`
- DON'T use Options API (use Composition API)

## Commands
```bash
cd front
pnpm install
pnpm dev          # http://localhost:3001
pnpm build        # Production build
pnpm exec nuxi typecheck
pnpm test:unit
```

## Notes
- No auth system (single-user app)
- UTF-8 only; never ANSI/GBK/UTF-16
- Preserve existing semicolon style per file
- UI has editorial/magazine feel (avoid generic SaaS look)
