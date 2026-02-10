# 数据库初始化模块

## 功能说明

数据库初始化模块会在启动时自动完成两件事：

1. **检查并创建缺失的表**（`EnsureTables()`）
2. **检查并记录数据库版本**（`EnsureSchemaVersion()`）

**自动执行时机：**
- 每次启动服务器时（`InitDB()` 中依次调用 `EnsureTables()` 和 `EnsureSchemaVersion()`）

**支持的表（共8张）：**
1. `categories` - 分类表
2. `feeds` - 订阅源表
3. `articles` - 文章表
4. `ai_summaries` - AI 摘要表
5. `scheduler_tasks` - 定时任务表
6. `ai_settings` - AI 设置表
7. `reading_behaviors` - 阅读行为表
8. `user_preferences` - 用户偏好表

## 工作原理

### 1. 表结构初始化（EnsureTables）

1. **检查表是否存在** - 查询 `sqlite_master` 判断表是否存在
2. **创建缺失的表** - 使用 `CREATE TABLE IF NOT EXISTS` 安全创建
3. **创建索引** - 自动创建所有必要的索引（`CREATE INDEX IF NOT EXISTS`）
4. **保护现有数据** - 不会修改或删除已存在的表和数据

### 2. 数据库版本管理（EnsureSchemaVersion）

新增了一套轻量级的 schema 版本管理，便于后续做增量迁移：

- 版本常量：`CurrentDBVersion`（当前 Go 后端期望的数据库版本，初始为 `1`）
- 版本记录表：`schema_migrations`

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  version INTEGER NOT NULL,
  applied_at DATETIME NOT NULL
);
```

**流程：**

1. 如果是 **sqlite**，`InitDB` 会在连接前判断数据库文件是否存在：
   - 不存在 → 视为「新数据库」，连接后自动创建文件
   - 存在 → 视为「已有数据库」
2. `EnsureSchemaVersion(isNewDB)` 会：
   - 确保创建 `schema_migrations` 表
   - 读取 `MAX(version)`，无记录时视为 `0`
   - 如果当前版本 `< CurrentDBVersion`，按版本号顺序执行迁移
3. 迁移逻辑集中在 `applyMigrations(fromVersion, toVersion)`：
   - 当前只定义了 `v1`，代表现有表结构的基线
   - 未来需要新增 schema 变更时，只需要：
     - 增大 `CurrentDBVersion`
     - 在 `switch v { ... }` 里新增对应 `case`，写入增量 SQL（ALTER TABLE / CREATE TABLE 等）
4. 每次成功迁移到新版本，都会在 `schema_migrations` 中插入一条记录，包含：
   - `version`：目标版本号
   - `applied_at`：执行时间

## 使用方式

### 自动初始化（推荐）

服务器启动时自动执行：
```bash
cd backend-go
go run cmd/server/main.go
```

### 手动检查

查看当前数据库中的表：
```bash
cd backend-go
go run cmd/list-tables/main.go
```

测试初始化功能：
```bash
cd backend-go
go run cmd/test-init/main.go
```

## 特性

✅ **安全**  
- 表初始化使用 `CREATE TABLE IF NOT EXISTS`  
- 版本表和迁移记录也使用显式 SQL，不依赖 AutoMigrate

✅ **幂等**  
- 表创建和索引创建均使用 `IF NOT EXISTS`  
- 版本迁移根据当前版本号判断，只会执行一次

✅ **可演进**  
- 使用 `schema_migrations` 记录版本，支持未来按版本做增量迁移  
- 可以同时兼容旧的 Python 后端数据库（旧库首次启动会被视为从 `0` 升级到 `1`）

✅ **日志输出**  
- 启动时会打印数据库文件是否存在、新建库或旧库升级的信息  
- 每次迁移到新版本都会输出对应日志，便于排查

## 示例输出

```
2025/02/05 15:15:02 Database initialized successfully
2025/02/05 15:15:02 Creating table: reading_behaviors
2025/02/05 15:15:02 ✓ Table created: reading_behaviors
```

## 注意事项

- 使用纯 SQL 语句创建表，避免 GORM AutoMigrate 的潜在问题
- 外键约束使用 `ON DELETE CASCADE` 确保数据一致性
- 索引使用 `IF NOT EXISTS` 确保重复执行不会报错
