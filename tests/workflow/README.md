# 工作流集成测试

测试后端核心调度器和数据处理流程。

## 测试覆盖范围

### 1. 调度器测试
- **Auto Refresh Scheduler**: 订阅源自动刷新
- **Firecrawl Scheduler**: 完整内容抓取
- **AI Summary Scheduler**: AI 总结生成
- **Auto Summary Scheduler**: 分类总结生成
- **Preference Update Scheduler**: 偏好更新

### 2. 工作流测试
- **完整流程测试**: Feed Refresh → Firecrawl → AI Summary
- **状态流转测试**: pending → processing → completed/failed
- **错误处理测试**: 各阶段失败场景
- **并发控制测试**: 信号量、批处理

### 3. 数据一致性测试
- **字段映射测试**: snake_case ↔ camelCase
- **状态一致性测试**: 状态机流转
- **外键约束测试**: Feed → Article

## 测试架构

```
tests/workflow/
├── README.md                    # 本文档
├── config.py                    # 测试配置
├── test_schedulers.py          # 调度器单元测试
├── test_workflow_integration.py # 完整工作流集成测试
├── test_data_flow.py           # 数据流转测试
├── test_error_handling.py      # 错误处理测试
└── utils/                      # 测试工具
    ├── __init__.py
    ├── database.py             # 数据库操作工具
    ├── api_client.py           # API 客户端
    └── mock_services.py        # 模拟服务
```

## 运行测试

### 前置要求
- Python 3.11+
- uv 包管理器
- 后端服务运行中 (localhost:5000)
- 数据库存在 (backend-go/rss_reader.db)

### 安装依赖
```bash
cd tests/workflow
uv venv
source .venv/bin/activate  # Windows: .venv\Scripts\activate
uv pip install requests pytest pytest-cov
```

### 运行所有测试
```bash
pytest test_*.py -v
```

### 运行特定测试
```bash
# 只测试调度器
pytest test_schedulers.py -v

# 只测试工作流
pytest test_workflow_integration.py -v

# 生成覆盖率报告
pytest --cov=. --cov-report=html
```

## 测试数据

测试使用真实的数据库，但会：
1. 创建临时测试数据
2. 测试完成后清理
3. 不影响生产数据

## 注意事项

1. **环境依赖**: 测试需要后端服务和 Firecrawl 服务运行
2. **测试隔离**: 每个测试独立运行，互不影响
3. **幂等性**: 测试可重复执行
4. **清理机制**: 测试完成后自动清理临时数据