# 打标签流程全景说明

> **版本**：基于 `backend-go/internal/domain/topicanalysis/` 与 `topicextraction/tagger.go` 实际代码整理  
> **阅读建议**：先看主流程图，再按需深入各子章节

---

## 1. 统一入口：`findOrCreateTag`

**触发场景**：文章/摘要生成标签、手动整理、叙事反馈等

```go
输入: tag topictypes.TopicTag, source string, articleContext string, articleID uint
输出: *models.TopicTag  (复用现有 / 新建 / 归入抽象)
```

### 主流程

```mermaid
flowchart TD
    START([新标签]) --> SVC{EmbeddingService<br/>可用?}
    SVC -->|否| FALLBACK[slug精确匹配<br/>或新建标签]
    SVC -->|是| TAGMATCH[TagMatch 三级匹配]
    
    TAGMATCH --> EXACT{MatchType}
    EXACT -->|exact| REUSE[复用现有标签<br/>更新元信息]
    EXACT -->|no_match| CREATE[新建标签<br/>生成embedding]
    EXACT -->|candidates| CAND{有候选<br/>≥0.78?}
    
    CAND -->|是 & event| COTAG[Co-tag扩展<br/>补充候选]
    CAND -->|否| CREATE
    COTAG --> LLM[送LLM批量判断]
    CAND -->|其他分类| LLM
    
    REUSE --> END1([返回标签])
    CREATE --> END1
    LLM --> RESULT{判断结果<br/>可部分覆盖候选}
    RESULT -->|merge| MERGE[合并到目标<br/>目标相似度须≥0.85]
    RESULT -->|abstract| ABSTRACT[创建抽象父标签]
    RESULT -->|merge+abstract| BOTH["MERGE后继续ABSTRACT<br/>其余候选视为none"]
    RESULT -->|none| CREATE
    MERGE --> END1
    ABSTRACT --> END1
    BOTH --> END1
```

---

## 2. Embedding 三级匹配：`TagMatch`

| 级别 | 匹配方式 | 阈值 | 行为 |
|------|----------|------|------|
| **L1** | slug 精确匹配 | — | 直接复用 |
| **L1** | 别名(alias)匹配 | — | 直接复用 |
| **L2** | embedding 相似搜索 | ≥0.97 | 自动复用(Exact) |
| **L2** | embedding 相似搜索 | 0.78~0.97 | 送LLM判断(Candidates) |
| **L2** | embedding 相似搜索 | <0.78 | 新建标签(No Match) |

---

## 3. LLM 批量判断：`callLLMForTagJudgment`

**输入**：候选列表(≤8个/批)、新标签名、分类、叙事上下文  
**输出**：`tagJudgment` (merge / abstract / merge+abstract / none)，部分覆盖即可，未覆盖的候选视为独立

```mermaid
sequenceDiagram
    participant Caller as findOrCreateTag
    participant LLM as LLM Router
    
    Caller->>Caller: 检查候选最高相似度<br/>≥0.85? 正常merge提示 : 加CAUTION警告
    Caller->>LLM: 构建Prompt(merge始终可用+含分类专属规则)
    Note right of Caller: person/event/keyword<br/>规则各不相同
    LLM-->>Caller: JSON响应<br/>{merge?: {...}, abstract?: {...}}
    Note right of Caller: merge和abstract可同时返回<br/>未覆盖的候选隐式为none<br/>例：候选A merge + BCD abstract + EF none
    Caller->>Caller: parseTagJudgmentResponse<br/>过滤无效候选、去重、截断
    Caller->>Caller: ensureNewLabelCandidateInAbstract<br/>确保newLabel在abstract.children中
```

### 判断规则速查

| 分类 | Merge条件 | Abstract条件 |
|------|-----------|--------------|
| **person** | 同一人物不同称谓 | 共享身份/机构/领域 |
| **event** | 同一事件不同描述 | 因果关联或同一事件链 |
| **keyword** | 同义词/翻译 | 同一具体领域直接相关 |

---

## 4. 抽象标签创建：`processAbstractJudgment`

**关键保护机制**：

```mermaid
flowchart LR
    A[开始创建抽象标签] --> B{slug与候选冲突?}
    B -->|是| C[跳过创建]
    B -->|否| D[findSimilarExistingAbstract<br/>查重已有抽象]
    D --> E{找到相同概念?}
    E -->|是| F[复用已有抽象]
    E -->|否| G[事务创建新抽象]
    G --> H{子标签数≥最小值?}
    H -->|普通路径| I[min=1<br/>+ newLabel作为第2个]
    H -->|整理/反馈路径| J[min=2<br/>已有候选≥2]
    H -->|不足| K[事务回滚<br/>errInsufficientAbstractChildren]
    I --> L[建立父子关系]
    J --> L
    F --> L
```

**异步副作用**：
- 生成 `identity` + `semantic` embedding
- 触发 `MatchAbstractTagHierarchy`
- `adoptNarrowerAbstractChildren` 收养更窄的抽象标签
- 入队 `abstract_tag_update_queues`

---

## 5. Event 标签 Co-tag 扩展

**目的**：用文章 keyword 反查共现 event，补充 embedding 召回遗漏

```mermaid
flowchart TD
    A[event标签有候选] --> B{来源}
    B -->|articleID| C1[取文章top5 keyword]
    B -->|abstractTagID| C2["聚合子树keyword覆盖<br/>topN = 5 + depth*2 - 2"]
    C1 --> D[反查共现文章]
    C2 --> D
    D --> E[提取关联event<br/>按hit_count排序]
    E --> F[与embedding候选并集<br/>相似度固定0.80]
```

---

## 6. 抽象层级匹配：`MatchAbstractTagHierarchy`

**触发**：新抽象标签创建后、刷新队列完成后

```mermaid
flowchart TD
    A[新抽象标签] --> B[Cross-layer Dedup]
    B --> C{高相似anchor<br/>≥0.97?}
    C -->|是| D[judgeCrossLayerDuplicate<br/>AI判断是否同一概念]
    D -->|是| E[MergeTags合并]
    D -->|否| F[继续层级匹配]
    C -->|否| F
    
    F --> G[FindSimilarAbstractTags]
    G --> H{相似度区间}
    H -->|≥0.97| I[mergeOrLinkSimilarAbstract<br/>merge/parent_A/parent_B]
    H -->|0.78~0.97| J{深度检查}
    J -->|childDepth+parentDepth+1 > 4| K[aiJudgeAlternativePlacement<br/>建议替代父标签]
    J -->|≤4| L[aiJudgeAbstractHierarchy<br/>AI判断谁更宽泛]
    L --> M[linkAbstractParentChild]
    H -->|<0.78| N[无操作]
```

---

## 7. 父子链接与多父冲突

### `linkAbstractParentChild` 保护

| 检查项 | 失败行为 |
|--------|----------|
| 循环检测 | 返回错误 |
| 深度限制(≤4层) | 触发AI建议替代位置 |
| 已存在关系 | 静默跳过 |

### `resolveMultiParentConflict`

```mermaid
flowchart LR
    A[子标签有多父?] --> B[removeRedundantAncestorParents<br/>检查祖先-后代]
    B -->|有祖先关系| C[删除祖先父<br/>保留更具体的]
    B -->|无祖先关系| D[aiJudgeBestParent<br/>AI选最佳归属]
    D --> E[删除其他父标签]
```

---

## 8. Embedding 保留策略

```mermaid
flowchart TD
    A[标签归入抽象父] --> B{父标签是抽象?}
    B -->|否| KEEP1[保留embedding<br/>独立标签]
    B -->|是| C{同级有抽象兄弟?}
    C -->|是| KEEP2[保留embedding<br/>需精确匹配锚点]
    C -->|否| DEL[删除embedding<br/>父抽象是唯一入口]
    
    KEEP1 --> END1([结束])
    KEEP2 --> END1
    DEL --> END1
    
    style KEEP1 fill:#e8f5e9,stroke:#2e7d32
    style KEEP2 fill:#e8f5e9,stroke:#2e7d32
    style DEL fill:#ffebee,stroke:#c62828
```

**动态补回**：当普通标签突然获得抽象兄弟时，`enqueueEmbeddingsForNormalChildren` 异步补生成 embedding。

---

## 9. 抽象标签刷新队列

**触发时机**：
- `new_child_added` — `ExtractAbstractTag` 完成
- `hierarchy_linked` — 建立抽象父子关系
- `tag_merged` — 合并到抽象标签
- `adopted_narrower_children` — 收养更窄子标签

```mermaid
flowchart TD
    TRIGGER([子标签变化触发入队]) --> |去重: 已有pending/processing则跳过| PENDING[pending]
    PENDING --> |worker 3s轮询 FOR UPDATE锁定| PROCESSING[processing]
    PROCESSING --> |LLM重生成label+desc → 检查slug冲突 → 生成identity+semantic embedding → 触发MatchAbstractTagHierarchy| COMPLETED[completed]
    PROCESSING --> |异常| FAILED[failed]
    FAILED --> |retry_count++ 可手动重试| DONE1([*])
    COMPLETED --> DONE2([*])

    style PENDING fill:#fff3e0,stroke:#e65100
    style PROCESSING fill:#e3f2fd,stroke:#1565c0
    style COMPLETED fill:#e8f5e9,stroke:#2e7d32
    style FAILED fill:#ffebee,stroke:#c62828
```

> 核心文件: `abstract_tag_update_queue.go`

---

## 10. 核心方法速查

| 方法 | 文件 | 输入 | 输出 | 一句话职责 |
|------|------|------|------|-----------|
| `findOrCreateTag` | `topicextraction/tagger.go` | tag, source, context, articleID | *TopicTag | **统一入口** |
| `TagMatch` | `topicanalysis/embedding.go` | label, category, aliases | TagMatchResult | **三级匹配** |
| `callLLMForTagJudgment` | `topicanalysis/abstract_tag_judgment.go` | candidates, newLabel, category, context | *tagJudgment | **LLM判断** |
| `processJudgment` | `topicanalysis/abstract_tag_service.go` | judgment, candidates, newLabel | TagExtractionResult | **结果处理** |
| `processAbstractJudgment` | `topicanalysis/abstract_tag_service.go` | candidates, judgment, newLabel, category | AbstractResult | **创建抽象标签** |
| `MatchAbstractTagHierarchy` | `topicanalysis/abstract_tag_hierarchy.go` | abstractTagID | - | **层级匹配** |
| `linkAbstractParentChild` | `topicanalysis/abstract_tag_hierarchy.go` | childID, parentID | error | **建立父子关系** |
| `resolveMultiParentConflict` | `topicanalysis/abstract_tag_hierarchy.go` | childID | bool | **解决多父冲突** |
| `ExpandEventCandidatesByArticleCoTags` | `topicanalysis/cotag_expansion.go` | articleID/abstractTagID | []TagCandidate | **co-tag扩展** |
| `refreshAbstractTag` | `topicanalysis/abstract_tag_update_queue.go` | abstractTagID | error | **刷新label/desc/emb** |
| `MergeTags` | `topicanalysis/embedding.go` | sourceID, targetID | error | **合并标签** |

---

## 11. 标签清理机制

### 实时清理：`cleanupOrphanedTags`

文章重新打标签后，旧标签若不再被任何 `article_topic_tags` 或 `ai_summary_topics` 引用，直接删除（含 embedding）。`article_tagger.go:369`

### 定时调度：`TagHierarchyCleanupScheduler`（4 阶段）

```mermaid
flowchart TD
    START([定时/手动触发]) --> P1[Phase 1: Zombie清理]
    P1 --> |无LLM 标记inactive| P2[Phase 2: Flat Merge]
    P2 --> |LLM判断 同类抽象标签合并 event/keyword各≤50| P3[Phase 3: 层级修剪]
    P3 --> P3A[删除孤儿关系]
    P3A --> P3B[解决多父冲突]
    P3B --> P3C[清理空抽象节点]
    P3C --> P4[Phase 4: 深度压缩]
    P4 --> P4A{深度≥3的标签树}
    P4A --> |跨层去重merge| P4B[LLM判断merge]
    P4A --> |深度>4| P4C[AI建议重新挂载]
    P4B --> END1([完成])
    P4C --> END1

    style P1 fill:#fff3e0
    style P2 fill:#e3f2fd
    style P3 fill:#f3e5f5
    style P4 fill:#e8f5e9
```

| 阶段 | 文件 | LLM? | 条件 |
|------|------|-------|------|
| Phase 1 Zombie | `tag_cleanup.go` `CleanupZombieTags` | 否 | age>7d + 无关系 + 无文章/摘要引用 |
| Phase 2 Flat Merge | `tag_cleanup.go` `ExecuteFlatMerge` | 是 | 同类 abstract 标签去重 |
| Phase 3 层级修剪 | `tag_cleanup.go` 三个函数 | 否 | 孤儿关系 / 多父 / 空抽象 |
| Phase 4 深度压缩 | `hierarchy_cleanup.go` `ExecuteHierarchyCleanupPhase4` | 是 | 深度≥3 的标签树 |

核心文件: `jobs/tag_hierarchy_cleanup.go`（调度器）、`topicanalysis/tag_cleanup.go`（Phase 1-3）、`topicanalysis/hierarchy_cleanup.go`（Phase 4）

---

## 总结

```
新标签 → Embedding三级匹配 → 有候选则LLM判断(merge/abstract/merge+abstract/none)
                                     ↓
                          event标签额外走co-tag扩展补充候选
                                     ↓
               LLM输出可同时包含merge和abstract: 高相似候选merge，其余候选abstract
                                     ↓
               merge: processJudgment验证目标相似度≥0.85 → 合并
               abstract: 查重 → 子标签数保护 → 事务落库
                                     ↓
               异步: 生成embedding → 层级匹配 → 收养更窄标签 → 入队刷新
                                     ↓
               层级匹配: C+D保护(跨层去重+深度限制) → AI判断父子
                                     ↓
               链接成功: 解决多父冲突 → embedding动态管理 → 父标签入队刷新
```
