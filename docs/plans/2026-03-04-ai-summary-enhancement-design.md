# AI总结增强功能设计文档

**日期:** 2026-03-04
**状态:** 设计阶段
**阶段:** Phase 1 (2周) + Phase 2 (4-6周)

---

## 1. 概述

### 1.1 目标

增强现有AI自动总结功能，提升新闻消费体验：

1. **定期简报** - 生成每日/每周汇总，便于快速浏览
2. **外部集成** - 支持飞书推送和Obsidian知识库
3. **智能关联** - 建立总结间的多维关联（Phase 2）

### 1.2 现状分析

**现有功能:**
- ✅ AutoSummaryScheduler - 定时生成各feed的AI总结
- ✅ AI总结格式 - 核心主题、重要新闻、核心观点、标签
- ✅ 前端展示 - AISummaryDetail组件

**痛点:**
- ❌ 总结分散，不便定期回顾
- ❌ 无外部推送和知识沉淀
- ❌ 缺少总结间关联

---

## 2. 架构设计

### 2.1 整体架构

```
现有: AutoSummaryScheduler → AI Service → Database
新增:
  └── DigestScheduler (统一处理日报/周报)
      ├── 生成每日/每周简报
      ├── FeishuNotifier推送 (可选)
      └── ObsidianExporter导出Markdown (可选)
```

### 2.2 数据流

**Phase 1:**
1. AutoSummaryScheduler生成AI总结 → 存入数据库
2. DigestScheduler定时触发:
   - **每日简报** (每天9点): 查询前一天的所有总结
     - 按分类组织汇总
     - 推送到飞书
     - 导出到Obsidian vault
   - **每周简报** (每周一9点): 查询上周所有总结
     - 按分类聚合
     - 推送到飞书
     - 导出到Obsidian vault

**Phase 2:**
3. Embedding存储和相似度计算
4. 实体抽取和知识图谱
5. 跨分类智能推荐

---

## 3. 功能设计

### 3.1 日报/周报生成

**时间调度:**
- 每日简报: 每天 09:00
- 每周简报: 每周一 09:00

**组织方式:**
- 按分类独立生成
- 每个分类包含该分类下所有订阅源的总结构

### 3.2 飞书推送

**推送策略:**

**① 汇总通知 (AI二次汇总)**
```
今日科技圈要点 📌

AI领域竞争白热化，OpenAI和Anthropic同天发布新一代模型
前端框架迎来重大更新，React 19和Vue 4候选版相继推出
技术社区关注AI工具集成，多个框架开始内置AI能力

📁 完整版已存 Obsidian → Daily/AI技术/2026-03-04-日报.md
```

**② 细节通知 (按feed推送)**
```
TechCrunch 今日新闻

## 核心主题
今天AI领域有两款重磅模型发布，OpenAI的GPT-5主打多模态推理...

## 🔥 热点事件
**OpenAI发布GPT-5**
模型在代码生成和数学推理上有显著提升，支持实时语音交互
> [OpenAI launches GPT-5 with enhanced reasoning](https://techcrunch.com/...)

**Apple发布会定档**
预计将发布iPhone 16系列和新iPad Pro...

## 相关标签
#TechCrunch #AI #Apple
```

**特点:**
- ✅ 口语化表达
- ✅ 直接引用原始文章链接
- ✅ 底部附Obsidian归档路径
- ✅ 简洁排版，手机友好

### 3.3 Obsidian集成

**文件结构:**
```
ObsidianVault/
├── Daily/
│   ├── AI技术/
│   │   └── 2026-03-04-日报.md
│   ├── 前端开发/
│   │   └── 2026-03-04-日报.md
│   └── 科技产品/
│       └── 2026-03-04-日报.md
├── Weekly/
│   ├── AI技术/
│   │   └── 2026-W9-周报.md
│   ├── 前端开发/
│   │   └── 2026-W9-周报.md
│   └── 科技产品/
│       └── 2026-W9-周报.md
└── Feeds/
    ├── TechCrunch/
    │   └── 2026-03-04.md
    ├── 阮一峰/
    │   └── 2026-03-04.md
    └── ...
```

**Markdown格式特点:**
- ✅ 书面化表达
- ✅ 包含YAML frontmatter
- ✅ Wiki链接支持双向关联
- ✅ 按分类组织

**Daily日报格式示例:**
```markdown
---
category: AI技术
date: 2026-03-04
type: daily-digest
tags: [daily, AI技术, 2026-03, 2026-W9]
---

# AI技术 - 2026年3月4日日报

## 本日概要

本日共生成3份订阅源总结，涵盖AI模型发布、行业动态、企业应用等方向。

## 订阅源总结

### TechCrunch

#### 核心主题
今日内容围绕AI模型发布和科技企业融资展开...

#### 重要事件
**OpenAI发布GPT-5**

模型在代码生成和数学推理能力上取得显著提升...
> [OpenAI launches GPT-5 with enhanced reasoning](https://techcrunch.com/xxx)

#### 核心观点
1. AI模型竞争从单一性能转向多模态综合能力
2. 企业级AI应用更注重安全性和可控性

### Hacker News
...

## 本日趋势

AI领域今日出现重大突破，两大模型同日发布...

## 相关链接

- [[Feeds/TechCrunch/2026-03-04]]
- [[Feeds/Hacker News/2026-03-04]]
```

### 3.4 基础关联 (Phase 1)

**支持的关联类型:**

1. **时间维度关联**
   - 同一feed的历史总结互相链接
   - 前后一天的总结链接

2. **同分类聚合**
   - 日报自动聚合同分类下的所有feed
   - 周报按分类汇总趋势

3. **标签关联**
   - 使用AI生成的标签进行关联
   - 在Obsidian中支持标签搜索

---

## 4. 高级AI功能 (Phase 2)

### 4.1 AI自动相似度匹配

**技术方案:**
- Embedding模型: OpenAI text-embedding-3-small
- 向量存储: SQLite向量扩展或独立向量库
- 相似度计算: 余弦相似度

**功能:**
- 自动推荐相似总结
- 在Obsidian中显示"相关内容"
- 跨时间的话题追踪

### 4.2 实体抽取和知识图谱

**技术方案:**
- NLP模型: GPT-4进行实体抽取
- 实体类型: 人名、公司、产品、事件

**功能:**
- 从文章中自动抽取实体
- 生成实体页面 (如 `[[OpenAI]]`, `[[GPT-5]]`)
- 建立实体关系网络
- 在Obsidian中构建知识图谱

### 4.3 跨分类智能推荐

**推荐策略:**
- 基于内容相似度
- 基于实体关联
- 基于用户阅读模式

**示例:**
AI技术分类的"GPT-5发布" → 推荐前端开发分类的"前端如何集成GPT-5"

---

## 5. 实现计划

### Phase 1: 基础功能 (2周)

**后端开发:**

**模块结构:**
```
backend-go/internal/digest/    -- 新增子模块
├── scheduler.go               -- DigestScheduler定时任务
├── generator.go               -- 日报/周报内容生成
├── feishu.go                  -- 飞书推送
├── obsidian.go                -- Obsidian导出
└── models.go                  -- 配置模型
```

**数据库表:**
```sql
CREATE TABLE digest_configs (
  id INTEGER PRIMARY KEY,
  daily_enabled BOOLEAN DEFAULT TRUE,
  daily_time VARCHAR(5) DEFAULT '09:00',
  weekly_enabled BOOLEAN DEFAULT TRUE,
  weekly_day VARCHAR(10) DEFAULT 'Monday',
  weekly_time VARCHAR(5) DEFAULT '09:00',
  feishu_enabled BOOLEAN DEFAULT TRUE,
  feishu_webhook_url VARCHAR(500),
  feishu_push_summary BOOLEAN DEFAULT TRUE,
  feishu_push_details BOOLEAN DEFAULT TRUE,
  obsidian_enabled BOOLEAN DEFAULT TRUE,
  obsidian_vault_path VARCHAR(1000),
  obsidian_daily_digest BOOLEAN DEFAULT TRUE,
  obsidian_weekly_digest BOOLEAN DEFAULT TRUE,
  created_at DATETIME,
  updated_at DATETIME
);
```

**API端点:**
```
GET  /api/digest/config          -- 获取配置
PUT  /api/digest/config          -- 更新配置
POST /api/digest/test-feishu     -- 测试飞书推送
POST /api/digest/test-obsidian   -- 测试Obsidian写入
GET  /api/digest/preview/:type   -- 预览日报/周报
GET  /api/digest/list            -- 获取日报/周报列表
GET  /api/digest/:id             -- 获取详情
```

**前端开发:**

**组件:**
```
front/app/components/layout/
└── SidebarContent.vue           -- 修改: 增加日报周报菜单

front/app/components/digest/     -- 新增
├── DigestPanel.vue             -- 日报/周报列表
├── DigestDetail.vue            -- 详情展示
└── DigestSettings.vue          -- 配置表单
```

**功能列表:**
- 日报/周报列表浏览
- 详情查看 (Markdown渲染)
- 配置管理界面
- 测试推送功能

### Phase 2: 高级AI功能 (4-6周)

**新增模块:**
```
backend-go/internal/digest/
├── embedding.go                -- Embedding生成和存储
├── entity_extractor.go         -- 实体抽取
├── knowledge_graph.go          -- 知识图谱构建
└── recommender.go              -- 智能推荐
```

**数据库表:**
```sql
-- 向量存储
CREATE TABLE summary_embeddings (
  id INTEGER PRIMARY KEY,
  summary_id INTEGER NOT NULL,
  embedding VECTOR(1536),        -- 需要SQLite向量扩展
  created_at DATETIME
);

-- 实体表
CREATE TABLE entities (
  id INTEGER PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  type VARCHAR(50),              -- person, company, product, event
  metadata JSON,
  created_at DATETIME
);

-- 实体关联表
CREATE TABLE summary_entities (
  summary_id INTEGER,
  entity_id INTEGER,
  mention_count INTEGER DEFAULT 1,
  PRIMARY KEY (summary_id, entity_id)
);
```

**功能:**
- Embedding自动生成
- 相似总结推荐API
- 实体自动抽取
- Obsidian实体页面生成
- 跨分类推荐

---

## 6. 错误处理

**飞书推送失败:**
- 记录日志到 `digest_logs` 表
- 不影响Obsidian导出
- 可配置重试次数

**Obsidian写入失败:**
- 记录详细错误信息
- 尝试写入到备份目录
- 在前端显示错误状态

**AI调用失败:**
- 标记为"生成失败"
- 日报中显示该feed暂无总结
- 可手动重试生成

---

## 7. 配置示例

**完整配置:**
```json
{
  "daily_enabled": true,
  "daily_time": "09:00",
  "weekly_enabled": true,
  "weekly_day": "Monday",
  "weekly_time": "09:00",
  "feishu_enabled": true,
  "feishu_webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
  "feishu_push_summary": true,
  "feishu_push_details": true,
  "obsidian_enabled": true,
  "obsidian_vault_path": "/Users/xxx/Documents/ObsidianVault",
  "obsidian_daily_digest": true,
  "obsidian_weekly_digest": true
}
```

---

## 8. 成功标准

**Phase 1:**
- ✅ 每天9点自动生成日报
- ✅ 每周一9点自动生成周报
- ✅ 飞书推送成功率达到95%+
- ✅ Obsidian文件格式正确，可正常预览
- ✅ 前端配置界面可用

**Phase 2:**
- ✅ 相似总结推荐准确率达到70%+
- ✅ 实体抽取覆盖率达到80%+
- ✅ 跨分类推荐相关度评分4.0+/5.0

---

## 9. 依赖关系

**Phase 1依赖:**
- 现有AutoSummaryScheduler正常运行
- 数据库可正常访问
- 文件系统写入权限

**Phase 2依赖:**
- Phase 1功能稳定运行
- OpenAI API支持embedding模型
- SQLite向量扩展或独立向量库

---

## 10. 后续优化方向

1. **多语言支持** - 英文feed的日报/周报
2. **自定义模板** - 用户自定义Markdown格式
3. **更多推送渠道** - 微信、Telegram、Email
4. **高级过滤** - 按重要性、主题过滤总结
5. **数据可视化** - 总结趋势图表
6. **协作功能** - 多用户共享知识库

---

## 附录

### A. 飞书Webhook设置指南

1. 在飞书群组中添加自定义机器人
2. 获取Webhook URL
3. 在应用配置中填入URL
4. 点击"测试推送"验证

### B. Obsidian Vault设置指南

1. 安装Obsidian
2. 创建或选择一个Vault
3. 确保应用有写入权限
4. 在配置中填入Vault完整路径
5. 点击"测试写入"验证

### C. Cron表达式说明

```
每日9点:    0 9 * * *
每周一9点:  0 9 * * 1
```
