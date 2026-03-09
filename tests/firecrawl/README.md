# Firecrawl集成测试

## 测试目标

验证Go后端的Firecrawl功能是否正常工作，包括：
- Firecrawl服务连接
- 数据库配置正确性
- 后端API功能
- 完整的文章抓取流程
- 抓取结果质量

## 运行测试

```bash
# 确保后端服务运行
cd backend-go
go run cmd/server/main.go

# 在另一个终端运行测试
cd tests/firecrawl
python test_firecrawl_integration.py
```

## 测试步骤

1. 配置验证 - 检查数据库中的Firecrawl配置
2. Firecrawl连接测试 - 直接测试Firecrawl API
3. 后端API测试 - 测试Go后端的Firecrawl相关API
4. 完整抓取流程 - 通过后端API触发完整抓取
5. 结果验证 - 验证抓取内容质量

## 扩展功能

测试脚本预留了扩展接口，支持未来添加：
- 去噪功能测试
- 内容摘要功能测试
- 批量抓取测试