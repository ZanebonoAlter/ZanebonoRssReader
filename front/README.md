# Frontend

前端基于 Nuxt 4、Vue 3、TypeScript 和 Pinia。

## 开发命令

```bash
pnpm install
pnpm dev
pnpm build
pnpm exec nuxi typecheck
```

## 当前入口

- 应用壳：`front/app/app.vue`
- 首页：`front/app/pages/index.vue`
- Digest 页面：`front/app/pages/digest/index.vue`

## 架构文档

- 前端架构：`docs/architecture/frontend.md`
- 数据流：`docs/architecture/data-flow.md`
- 开发流程：`docs/operations/development.md`

## 目录重组方向

前端将从“按技术层堆目录”逐步迁移到“按 feature 组织”的结构，核心目标是：

- `api/` 成为唯一接口边界
- `features/` 收拢业务代码
- `shared/` 只放通用能力
