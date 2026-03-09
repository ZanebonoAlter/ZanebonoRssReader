# 排障

## 前端起不来

- 确认 `front/` 里依赖已安装
- 运行 `pnpm exec nuxi typecheck`
- 检查是否有编码问题或错误 import

## 后端起不来

- 检查 `backend-go/configs/config.yaml`
- 运行 `go test ./...`
- 检查数据库文件路径和端口占用

## 文档又漂移了

- 从 `README.md` 和 `docs/README.md` 开始核对
- 删除失效路径引用
- 不要在根目录继续新增一次性说明文档
