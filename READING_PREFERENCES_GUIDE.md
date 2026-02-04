# 阅读行为评估功能 - 快速指南

## 功能概述

阅读行为评估系统会自动追踪您的阅读习惯，用于 AI 摘要个性化。

## 核心功能

### 1. 自动追踪
- ✅ 文章打开/关闭事件
- ✅ 滚动深度（0-100%）
- ✅ 阅读时长（秒）
- ✅ 收藏/取消收藏操作
- ✅ 每 30 秒自动上传
- ✅ 批量上传优化性能

### 2. 偏好分析
- ✅ 按订阅源计算偏好分数
- ✅ 按分类计算偏好分数
- ✅ 多维度评分算法
- ✅ 时间衰减（30天半衰期）
- ✅ 每 30 分钟自动更新

### 3. 可视化展示
- ✅ 集成到全局设置 → 阅读偏好
- ✅ 实时统计数据展示
- ✅ 偏好分数可视化
- ✅ 支持手动更新

## 使用方法

### 查看阅读偏好

1. 点击右上角设置按钮
2. 切换到"阅读偏好"标签
3. 查看：
   - 总文章数、总阅读时长
   - 平均阅读时长、平均滚动深度
   - 按订阅源/分类的偏好分数

### 偏好分数说明

**分数范围**：0-100%

**计算公式**：
```
分数 = (滚动深度 × 40% + 阅读时长 × 30% + 互动频率 × 30%) 
      × 时间衰减因子
```

**分数含义**：
- **70%+**：非常感兴趣（绿色）
- **40-70%**：比较感兴趣（黄色）
- **<40%**：一般兴趣（灰色）

## 数据隐私

- 所有数据存储在本地数据库
- 无用户标识，使用匿名 session_id
- 仅用于改善个人阅读体验
- 可随时删除追踪数据

## API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/reading-behavior/track` | POST | 记录单个行为 |
| `/api/reading-behavior/track-batch` | POST | 批量记录行为 |
| `/api/reading-behavior/stats` | GET | 获取统计信息 |
| `/api/user-preferences` | GET | 获取用户偏好 |
| `/api/user-preferences/update` | POST | 手动触发更新 |

## 数据流程

```
阅读文章
  ↓
自动追踪（ArticleContent.vue）
  ↓
批量上传（每30秒）
  ↓
后端存储（reading_behaviors 表）
  ↓
定时聚合（每30分钟）
  ↓
偏好计算（user_preferences 表）
  ↓
可视化展示（全局设置）
```

## 技术实现

### 后端
- **模型**：ReadingBehavior, UserPreference
- **服务**：PreferenceService, AISummaryPromptBuilder
- **调度器**：PreferenceUpdateScheduler (30分钟)
- **算法**：多维度加权 + 时间衰减

### 前端
- **追踪**：useReadingTracker composable
- **状态**：usePreferencesStore (Pinia)
- **API**：reading_behavior.ts
- **UI**：GlobalSettingsDialog (阅读偏好面板)

## AI 集成

偏好数据用于：
1. **个性化摘要** - 根据偏好调整摘要风格和重点
2. **推荐排序** - 基于偏好分数对文章排序（待实现）
3. **内容优化** - 优先展示用户感兴趣的内容

## 文件清单

### 后端新增/修改
```
backend-go/
├── internal/
│   ├── models/
│   │   ├── reading_behavior.go       [新增]
│   │   └── user_preference.go        [新增]
│   ├── handlers/
│   │   └── reading_behavior.go       [新增]
│   ├── services/
│   │   ├── preference_service.go     [新增]
│   │   └── ai_prompt_builder.go      [新增]
│   └── schedulers/
│       └── preference_update.go      [新增]
├── cmd/
│   ├── create-behavior-tables/       [新增]
│   └── server/main.go                [修改]
└── pkg/database/db.go                [修改]
```

### 前端新增/修改
```
front/app/
├── components/
│   ├── article/
│   │   └── ArticleContent.vue         [修改 - 集成追踪]
│   └── dialog/
│       └── GlobalSettingsDialog.vue   [修改 - 添加偏好面板]
├── composables/
│   ├── api/
│   │   └── reading_behavior.ts        [新增]
│   └── useReadingTracker.ts           [新增]
├── stores/
│   └── preferences.ts                 [新增]
└── types/
    ├── reading_behavior.ts            [新增]
    └── index.ts                       [修改]
```

### 文档更新
- `backend-go/ARCHITECTURE.md` - 添加新模块说明
- `front/ARCHITECTURE.md` - 添加新功能说明

## 常见问题

**Q: 如何重置偏好数据？**
A: 直接删除数据库中的 reading_behaviors 和 user_preferences 表数据即可。

**Q: 偏好更新太慢怎么办？**
A: 在全局设置 → 阅读偏好中点击"更新偏好"按钮手动触发。

**Q: 如何禁用追踪？**
A: 移除 ArticleContent.vue 中的 useReadingTracker 相关代码即可。

**Q: 数据准确性如何？**
A: 基于 3 个维度综合计算，时间衰减确保偏好时效性。

---

**版本**: 1.0.0  
**更新日期**: 2026-02-04  
**状态**: ✅ 已完成并集成到全局设置
