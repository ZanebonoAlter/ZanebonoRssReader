# 跨分类叙事整合设计

## 问题

当前全局叙事（`scope_type = global`）直接从标签采集生成，输入是所有根抽象节点 + 未分类活跃标签（上限 100 个）。随着标签体系膨胀，存在：

1. LLM prompt 过长，生成质量下降
2. 叙事过于发散，关联性弱
3. 与分类叙事职责重叠

## 方案

全局叙事改为**分类叙事摘要的二次整合**，只负责发现跨分类的关联线索。

### 生成流程

```
分类叙事（各 category 独立生成，已有）
           ↓
  CollectCategoryNarrativeSummaries
           ↓
  GenerateCrossCategoryNarratives
           ↓
  全局叙事（scope_type = global）
```

严格顺序：先完成所有分类叙事，再以分类叙事摘要为输入生成全局叙事。

### 改动范围

#### 后端 `collector.go`

新增 `CollectCategoryNarrativeSummaries(date)` 函数：
- 查询当日 `scope_type = 'feed_category'` 的所有叙事
- 按 category 分组，构建结构化输入：
  ```
  CategoryInput {
    CategoryName    string
    CategoryIcon    string
    Narratives      []CategoryNarrativeBrief
  }
  CategoryNarrativeBrief {
    ID             uint64
    Title          string
    Summary        string
    RelatedTags    []TagBrief
  }
  ```

#### 后端 `generator.go`

新增 `GenerateCrossCategoryNarratives(ctx, categoryInputs, prevGlobalNarratives)` 函数：
- 独立的 system prompt，明确要求只输出横跨≥2个分类的关联叙事
- JSON schema 与现有 `NarrativeOutput` 兼容，额外输出 `source_category_ids`
- `parent_ids` 指向分类叙事 ID

#### 后端 `service.go`

修改 `GenerateAndSave(date)`：
1. 先调用 `GenerateAndSaveForAllCategories(date)` 生成分类叙事
2. 调用 `CollectCategoryNarrativeSummaries(date)` 收集摘要
3. 如果分类叙事为 0，跳过全局叙事
4. 调用 `GenerateCrossCategoryNarratives` 生成全局叙事
5. 保存时 `scope_type = global`，`parent_ids` 指向分类叙事

修改 `RegenerateAndSave(date)`：
- 删除+重建的顺序不变，内部逻辑随 `GenerateAndSave` 一起变

#### 数据模型

`NarrativeSummary` 无字段变更。全局叙事的 `parent_ids` 现在指向分类叙事 ID，历史链路自然串联。

#### 前端

`NarrativePanel.vue` 无需改动。`全部/按分类` 切换已实现，全局叙事的 parent 关系可被历史面板正常展示。

### Prompt 设计

全局叙事 system prompt 核心要求：

```
你是一名专业的新闻叙事分析师。你收到了各分类频道独立生成的叙事摘要。
你的任务是发现横跨多个分类频道的关联叙事线索。

规则：
1. 只输出横跨至少 2 个分类的叙事
2. 每条叙事标注来源分类
3. 标题必须是带判断的短句，不超过 30 字
4. 摘要 200-400 字，说明跨分类的因果/影响/主题关联
5. 不要重复分类内部已发现的叙事
6. 数量不固定，没有跨分类关联就返回空数组
```

### 向后兼容

- `CollectTagInputs` 和 `GenerateNarratives` 保留，不删除，用于分类叙事
- 现有 API 端点不变
- 前端查询逻辑不变（`scope_type` 过滤依然生效）

### 调度变更

`NarrativeSummaryScheduler` 调用 `GenerateAndSave` 时，内部自动先跑分类再跑全局，调度器代码无需改动。
