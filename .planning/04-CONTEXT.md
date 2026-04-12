# Phase 04: API规范化 CONTEXT

## Overview
目标：统一前端API调用格式，修复状态同步问题，确保前后端交互一致性。
所属里程碑：v1.1 业务漏洞修复
依赖：Phase 01-03已完成，无阻塞依赖。

## Requirements to Implement
| ID | 需求描述 | 完成标准 |
|----|----------|----------|
| API-01 | Scheduler trigger API统一使用apiClient，不直接使用fetch | 前端scheduler.ts中的triggerSchedulerRequest方法移除fetch调用，改为使用apiClient.post，返回格式与其他API方法一致 |
| API-02 | UpdateArticle成功后正确更新相关feed的unreadCount | 1. 标记文章为已读时，对应feed.unreadCount减1<br>2. 标记文章为未读时，对应feed.unreadCount加1<br>3. 防止计数出现负数 |
| API-03 | MarkAllAsRead的本地状态更新覆盖所有边界情况 | 1. 全量标记已读、按feed标记已读、按分类标记已读、标记未分类已读四种场景下，对应feed的unreadCount都正确清零<br>2. 不影响非目标feed的计数 |
| API-04 | 所有scheduler status API返回格式一致 | 所有scheduler的GetStatus方法都返回以下字段：<br>- name: string (调度器名称)<br>- status: string (运行状态: running/idle/stopped/error)<br>- check_interval: int (检查间隔，单位秒)<br>- next_run: string (下次执行时间，RFC3339格式)<br>- is_executing: boolean (是否正在执行)<br>（可选附加字段可保留，但必须包含以上必填字段） |

## Decisions Made
1. **API调用方式**：triggerScheduler改用apiClient.post，复用现有apiClient的错误处理和返回格式，保持代码一致性。
2. **状态更新策略**：前端本地直接更新unreadCount，无需重新拉取全量feed数据，提升响应速度。
3. **后端一致性保障**：创建共享SchedulerStatus struct定义标准返回字段，所有scheduler的GetStatus方法都返回该struct转换的map，避免未来出现不一致。
4. **边界处理**：
   - updateArticle处理read=true和read=false两种情况，分别增减计数
   - markAllAsRead按操作范围（全量/feed/分类/未分类）遍历对应feed，将unreadCount设为0
   - 计数操作增加最小值0的保护，防止出现负数

## Implementation Details
### Frontend Changes
#### 1. front/app/api/scheduler.ts
- 移除triggerSchedulerRequest函数中的fetch调用
- 改为使用`apiClient.post<SchedulerTriggerResult>(`/schedulers/${name}/trigger`, {})`
- 保持返回格式不变，与现有apiClient调用风格一致

#### 2. front/app/stores/api.ts
- **updateArticle函数**：
  - 修改lines 275-281逻辑，增加read=false的处理：
    - 当data.read === true且article.read === false时，feed.unreadCount--
    - 当data.read === false且article.read === true时，feed.unreadCount++
  - 增加边界保护：计数不能小于0
- **markAllAsRead函数**：
  - 在lines 317-332更新article.read状态后，增加对应feed计数更新逻辑：
    - 全量标记：遍历所有feed，将unreadCount设为0
    - 按feed标记：找到指定feed，将unreadCount设为0
    - 按分类标记：找到该分类下所有feed，将unreadCount设为0
    - 未分类标记：找到所有无category的feed，将unreadCount设为0

### Backend Changes
#### 1. backend-go/internal/jobs/common.go (新建文件)
- 定义SchedulerStatus标准结构体：
  ```go
  type SchedulerStatus struct {
      Name          string      `json:"name"`
      Status        string      `json:"status"`
      CheckInterval int         `json:"check_interval"`
      NextRun       string      `json:"next_run"`
      IsExecuting   bool        `json:"is_executing"`
      // 可选附加字段
      Extra         map[string]interface{} `json:"extra,omitempty"`
  }
  ```
- 提供ToMap()方法转换为gin.H格式

#### 2. 所有scheduler实现文件更新GetStatus方法
- auto_refresh.go：已有全部必填字段，可继续使用原有逻辑，或迁移到新struct
- firecrawl.go：
  - 增加next_run字段（从cron entries或scheduler task获取）
  - 增加is_executing字段（使用atomic布尔值标记执行状态）
  - 现有字段（queue_size, processing等）移入Extra字段保留
- 其他scheduler（auto_summary, preference_update, content_completion, digest）：
  - 检查并补充缺失的必填字段
  - 统一使用SchedulerStatus struct返回

## Files to Modify
### Frontend
- `front/app/api/scheduler.ts` (API-01)
- `front/app/stores/api.ts` (API-02, API-03)

### Backend
- `backend-go/internal/jobs/auto_refresh.go` (API-04)
- `backend-go/internal/jobs/firecrawl.go` (API-04)
- `backend-go/internal/jobs/auto_summary.go` (API-04)
- `backend-go/internal/jobs/preference_update.go` (API-04)
- `backend-go/internal/jobs/content_completion.go` (API-04)
- `backend-go/internal/jobs/digest.go` (API-04)
- `backend-go/internal/jobs/common.go` (新增，API-04)

## Verification Steps
1. **API-01验证**：
   - 点击任意调度器的"立即执行"按钮
   - 检查网络请求使用apiClient格式，无直接fetch调用
   - 调度器正确触发，返回结果正常显示

2. **API-02验证**：
   - 选择一篇未读文章，标记为已读，对应feed未读计数减1
   - 再标记为未读，对应feed未读计数加1
   - 计数不会出现负数

3. **API-03验证**：
   - 全量标记已读：所有feed未读计数清零
   - 按单个feed标记已读：仅该feed计数清零
   - 按分类标记已读：该分类下所有feed计数清零
   - 标记未分类已读：所有无分类feed计数清零

4. **API-04验证**：
   - 调用GET /api/schedulers/status接口
   - 检查所有返回的调度器对象都包含name, status, check_interval, next_run, is_executing字段
   - 字段类型正确，无缺失

## Out of Scope
- 不修改API路径和现有请求/返回格式（除统一必填字段外）
- 不添加新的API端点
- 不修改UI展示逻辑，仅修复状态同步问题
- 不涉及后端定时任务执行逻辑修改，仅调整状态返回格式
