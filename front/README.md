# RSS Reader Frontend

基于 **Nuxt 4** + **Vue 3** + **TypeScript** 的现代化 RSS 阅读器前端应用。

采用 **Editorial/Magazine（杂志/报纸）设计风格**，为内容消费打造沉浸式阅读体验。

---

## 🎯 快速开始

### 环境要求

- Node.js 18+
- pnpm 10.15.0+
- 后端服务运行于 `localhost:5000`

### 安装依赖

```bash
pnpm install
```

### 启动开发服务器

```bash
pnpm dev
```

应用将运行于 `http://localhost:3001`

### 生产构建

```bash
pnpm build
pnpm preview
```

---

## 📚 项目文档

本项目包含详细的技术文档，建议按顺序阅读：

### 📖 必读文档

1. **[ARCHITECTURE.md](./ARCHITECTURE.md)** ⭐
   
   项目架构完整文档
   - 技术栈详解
   - 目录结构说明
   - API 层设计
   - 状态管理架构
   - 数据流向说明
   - 设计系统规范

2. **[COLOR_SYSTEM.md](./COLOR_SYSTEM.md)** 🎨
   
   颜色与样式系统设计文档
   - Editorial/Magazine 设计理念
   - 完整颜色变量表
   - 组件样式规范
   - 使用指南和最佳实践

### 🔧 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Nuxt | ^4.2.2 | 核心框架 |
| Vue | ^3.5.26 | Vue 核心 |
| TypeScript | - | 类型安全 |
| Tailwind CSS | ^4.1.18 | 样式框架 |
| Pinia | ^3.0.4 | 状态管理 |
| @iconify/vue | ^5.0.0 | 图标组件 |
| Day.js | ^1.11.19 | 日期处理 |
| Marked | ^17.0.1 | Markdown 渲染 |

---

## 🏗️ 项目结构

```
front/
├── app/
│   ├── app.vue                 # 根组件
│   ├── components/             # Vue 组件
│   │   ├── layout/             # 布局组件
│   │   ├── article/            # 文章组件
│   │   ├── ai/                 # AI 组件
│   │   ├── dialog/             # 对话框组件
│   │   └── ...
│   ├── composables/           # 组合式函数
│   │   └── api/               # API 层
│   ├── stores/                # Pinia 状态管理
│   ├── types/                 # TypeScript 类型
│   ├── utils/                 # 工具函数
│   └── assets/css/           # 全局样式
├── ARCHITECTURE.md            # 架构文档 ⭐
├── COLOR_SYSTEM.md            # 颜色系统文档 ⭐
└── README.md                  # 本文件
```

详细目录结构说明请查看 [ARCHITECTURE.md](./ARCHITECTURE.md)。

---

## 🎨 设计系统

本项目采用 **Editorial/Magazine（杂志/报纸）设计风格**，与其他 SaaS 应用区分开来。

### 核心特征

- ✨ **温暖纸张色调** - 象牙白、奶油色背景
- 🎯 **高对比度排版** - 优秀的可读性
- 🖋️ **印刷品配色** - 墨水蓝 + 印刷红
- 📰 **杂志质感** - 优雅阴影和纹理

### 颜色速查

| 用途 | 颜色 | CSS 变量 |
|------|------|----------|
| 主色 | 墨水蓝 `#3b6b87` | `--color-ink-500` |
| 强调色 | 印刷红 `#d94a4a` | `--color-print-red-500` |
| 背景 | 象牙白 `#faf7f2` | `--color-paper-ivory` |
| 文字 | 墨黑 `#1a1a1a` | `--color-ink-black` |

完整颜色系统请查看 [COLOR_SYSTEM.md](./COLOR_SYSTEM.md)。

---

## 🚀 开发指南

### 添加新组件

1. 在 `app/components/` 对应目录创建 `.vue` 文件
2. 使用 `<script setup lang="ts">` 语法
3. 遵循命名约定：PascalCase（如 `FeedLayout.vue`）

```vue
<script setup lang="ts">
import { ref } from 'vue'

interface Props {
  title: string
}

const props = defineProps<Props>()
</script>

<template>
  <div class="paper-card">
    {{ title }}
  </div>
</template>
```

### 添加新 API 端点

1. 在 `app/composables/api/` 创建模块
2. 使用 `ApiClient` 封装请求
3. 在 `index.ts` 导出

```typescript
// app/composables/api/myFeature.ts
export function useMyFeatureApi() {
  return {
    async getData() {
      return apiClient.get('/my-feature')
    }
  }
}
```

### 样式最佳实践

- ✅ 使用 CSS 变量而非硬编码颜色
- ✅ 优先使用 Tailwind 工具类
- ✅ 组件特定样式放在 `<style scoped>` 中
- ✅ 复杂动画使用 `@keyframes` 定义

```vue
<style scoped>
.my-component {
  background: var(--color-paper-cream);
  color: var(--color-ink-dark);
}
</style>
```

---

## 🔗 与后端集成

### API 基础配置

- **Base URL**: `http://localhost:5000/api`
- **响应格式**: `{ success: boolean, data?: T, error?: string }`

### 核心端点

| 功能 | 端点 | 方法 |
|------|------|------|
| 分类 | /api/categories | GET/POST/PUT/DELETE |
| 订阅源 | /api/feeds | GET/POST/PUT/DELETE |
| 文章 | /api/articles | GET/PUT |
| AI 摘要 | /api/summaries | GET/POST/DELETE |
| OPML | /api/import-opml | POST |

详细 API 文档请查看 [ARCHITECTURE.md](./ARCHITECTURE.md)。

---

## 🐛 常见问题

### 端口冲突

如果 3001 端口被占用，修改 `nuxt.config.ts`：

```typescript
export default defineNuxtConfig({
  devServer: {
    port: 3002 // 或其他端口
  }
})
```

### 样式不生效

1. 清除 `.nuxt` 缓存：`rm -rf .nuxt`
2. 重启开发服务器：`pnpm dev`
3. 强制刷新浏览器：`Ctrl + Shift + R` (Windows) / `Cmd + Shift + R` (Mac)

### 类型错误

运行类型检查：

```bash
npx nuxi typecheck
```

---

## 📝 开发命令

```bash
# 安装依赖
pnpm install

# 启动开发服务器 (localhost:3001)
pnpm dev

# 类型检查
npx nuxi typecheck

# 生产构建
pnpm build

# 预览生产构建
pnpm preview
```

---

## 🤝 贡献指南

### 代码风格

- 使用 TypeScript 编写所有新代码
- 遵循现有的文件组织结构
- 组件使用 Composition API + `<script setup>`
- 样式优先使用 Tailwind CSS 工具类

### 提交规范

提交前请确保：
- ✅ 代码通过类型检查 (`npx nuxi typecheck`)
- ✅ 遵循颜色系统规范 (参考 COLOR_SYSTEM.md)
- ✅ 组件样式与整体设计一致

---

## 📄 许可证

本项目为个人学习项目，仅供研究参考。

---

## 🔗 相关链接

- [Nuxt 4 文档](https://nuxt.com/docs)
- [Vue 3 文档](https://vuejs.org)
- [Tailwind CSS v4 文档](https://tailwindcss.com/docs)
- [Pinia 文档](https://pinia.vuejs.org)
- [Iconify 图标库](https://iconify.design)

---

**版本**: 2.0  
**最后更新**: 2026-02-05  
**设计系统**: Editorial/Magazine Theme
