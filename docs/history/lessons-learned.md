# Lessons Learned

这份文档记录已经反复踩过的坑，重点是减少重复调试和重复返工。

## 后端

- Gin 请求体不要既 `GetRawData()` 又 `ShouldBindJSON()`
- 新字段加入响应后，前端映射要同步改

## 前端

- props 初始化成 `ref` 后不会自动同步
- `v-model` 需要明确默认值
- snake_case 到 camelCase 的转换只放在 API 边界

## 编码安全

- Windows 下整文件 shell 重写很容易把中文写坏
- 发现乱码后别一点点修，直接整文件重写
