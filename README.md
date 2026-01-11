# RSS 订阅管理系统

一个功能强大的 RSS 订阅管理系统，支持 AI 智能分析、分类管理、自动刷新和文章阅读。

## 界面预览

### 📰 文章阅读界面
清爽的三栏布局设计，左侧分类导航，中间文章列表，右侧文章内容阅读区。

![文章阅读界面](img/image_article.png)

### 🤖 AI 智能摘要
AI 自动分析文章内容，提取核心观点、关键要点并生成标签。

![AI 摘要功能](img/img_ai.png)

### ⚙️ 全局设置
支持 AI 服务配置、定时任务管理、OPML 导入导出等功能。

![设置界面](img/image_setting.png)

## 项目结构

```
my-robot/
├── backend/                 # Python Flask 后端
│   ├── app.py              # Flask 应用入口，蓝图注册
│   ├── config.py           # 应用配置（CORS 等）
│   ├── database.py         # SQLite 数据库初始化
│   ├── models.py           # SQLAlchemy 数据模型
│   ├── rss_parser.py       # RSS/Atom 订阅源解析
│   ├── scheduler_base.py   # 定时任务基类
│   ├── auto_refresh.py     # 自动刷新调度器
│   ├── auto_summary.py     # AI 自动摘要调度器
│   ├── init_data.py        # 初始化数据
│   ├── tasks.py            # 后台任务管理
│   ├── routes/             # API 路由蓝图
│   │   ├── categories.py   # /api/categories - 分类 CRUD
│   │   ├── feeds.py        # /api/feeds - 订阅源管理
│   │   ├── articles.py     # /api/articles - 文章查询和状态
│   │   ├── ai.py           # /api/ai - AI 智能分析
│   │   ├── summaries.py    # /api/summaries - AI 摘要管理
│   │   ├── schedulers.py   # /api/schedulers - 定时任务控制
│   │   └── opml.py         # /api/opml - OPML 导入导出
│   ├── requirements.txt    # Python 依赖（UV 生成）
│   └── rss_reader.db       # SQLite 数据库文件
│
└── front/                   # Nuxt 4 前端
    ├── app/                # Nuxt 应用目录
    │   ├── app.vue         # 根组件
    │   ├── components/     # Vue 组件
    │   │   ├── Crud/       # CRUD 对话框组件
    │   │   ├── summary/    # AI 摘要相关组件
    │   │   ├── ArticleCard.vue
    │   │   ├── CategoryCard.vue
    │   │   ├── FeedIcon.vue
    │   │   ├── FeedLayout.vue  # 三栏布局组件
    │   │   └── GlobalSettingsDialog.vue
    │   ├── composables/    # 组合式函数
    │   │   ├── useApi.ts       # API 客户端
    │   │   ├── useAI.ts        # AI 功能
    │   │   ├── useAutoRefresh.ts  # 自动刷新
    │   │   └── useRssParser.ts    # RSS 解析
    │   ├── pages/          # 页面路由
    │   │   ├── index.vue        # 首页
    │   │   └── article/
    │   │       └── [id].vue     # 文章详情页
    │   ├── plugins/        # Nuxt 插件
    │   ├── stores/         # Pinia 状态管理
    │   │   ├── api.ts      # API Store
    │   │   ├── feeds.ts    # 订阅源 Store
    │   │   └── articles.ts # 文章 Store
    │   └── types/          # TypeScript 类型定义
    ├── nuxt.config.ts      # Nuxt 配置
    ├── package.json        # 依赖管理
    └── tsconfig.json       # TypeScript 配置
```

## 功能特性

### AI 智能分析
- **文章智能总结**: 使用 AI 对单篇文章进行智能摘要
  - 一句话总结
  - 核心观点提取
  - 关键要点归纳
  - 自动标签生成
- **分类聚合摘要**: 按分类对多篇文章进行聚合分析
  - 支持自定义时间范围
  - 自动提取核心趋势
  - 跨文章主题关联
- **兼容多种 AI 服务**: 支持 OpenAI 及兼容 API
- **定时自动摘要**: 自动为新增文章生成 AI 摘要
- **AI 配置管理**: 灵活配置 AI 服务参数

### RSS 订阅管理
- 添加/删除 RSS 订阅源
- 订阅源预览功能
- 自动获取 RSS 文章
- 订阅源分类管理
- **自动刷新**: 支持定时自动刷新订阅源
  - 自定义刷新间隔
  - 刷新状态监控
  - 错误日志记录
- **文章数量控制**: 每个订阅源可设置最大文章保留数量

### 文章阅读
- 文章列表展示
- 已读/未读标记
- 收藏文章功能
- 文章内容阅读
- 在原网站查看

### 分类功能
- 创建/编辑/删除分类
- 按分类查看订阅
- 分类统计信息

### 数据导入导出
- OPML 格式导入
- OPML 格式导出
- 批量管理订阅源

### 定时任务
- 自动刷新调度器
- AI 摘要生成调度器
- 任务执行状态监控
- 执行历史和错误日志

## 技术栈

### 后端
- **Python**: >= 3.13
- **Flask**: Web 框架
- **SQLAlchemy**: ORM 数据库操作
- **SQLite**: 轻量级数据库
- **feedparser**: RSS/Atom 解析
- **Flask-CORS**: 跨域支持
- **requests**: HTTP 请求
- **crawl4ai**: 网页内容提取（可选）
- **beautifulsoup4**: HTML 解析

### 前端
- **Nuxt 4**: Vue.js 全栈框架
- **Pinia**: 状态管理
- **TailwindCSS**: 样式框架
- **TypeScript**: 类型安全
- **VueUse**: Vue 组合式工具集

## 安装和运行

### 前置要求

- **Python**: >= 3.13
- **Node.js**: 推荐 18.x 或更高版本
- **pnpm**: 前端包管理器
- **UV**: Python 包管理器（推荐）

### 后端设置

#### 方法一：使用 UV（推荐）

UV 是一个快速的 Python 包管理器，可以自动管理虚拟环境和依赖。

1. 安装 UV（如果尚未安装）：
```bash
# Windows (PowerShell)
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"

# macOS/Linux
curl -LsSf https://astral.sh/uv/install.sh | sh
```

2. 同步依赖并创建虚拟环境：
```bash
cd backend
uv sync
```

3. 激活虚拟环境并启动服务器：
```bash
# Windows
.venv\Scripts\activate
python app.py

# macOS/Linux
source .venv/bin/activate
python app.py
```

服务器将在 `http://localhost:5000` 启动

#### 方法二：使用传统 pip

1. 进入后端目录并创建虚拟环境：
```bash
cd backend
python -m venv venv
```

2. 激活虚拟环境：
```bash
# Windows
venv\Scripts\activate

# macOS/Linux
source venv/bin/activate
```

3. 安装依赖：
```bash
pip install -r requirements.txt
```

4. 启动Flask服务器：
```bash
python app.py
```

服务器将在 `http://localhost:5000` 启动

### 前端设置

1. 进入前端目录：
```bash
cd front
```

2. 安装依赖：
```bash
pnpm install
```

3. 启动开发服务器：
```bash
pnpm dev
```

前端将在 `http://localhost:3000` 启动

**注意**：
- 如果遇到 CSS 相关错误，尝试清理缓存：`rm -rf .nuxt node_modules && pnpm install`
- 前端默认连接到后端 `http://localhost:5000/api`

## 使用说明

### 基本使用

1. 启动后端和前端服务
2. 访问 `http://localhost:3000`
3. 创建分类并添加 RSS 订阅源
4. 系统将自动抓取文章

### AI 功能配置

1. 点击设置中的 "AI 配置"
2. 填写 AI 服务信息：
   - **API 地址**: 如 `https://api.openai.com/v1`
   - **API Key**: 您的 API 密钥
   - **模型**: 如 `gpt-4o-mini`
3. 测试连接并保存
4. 启用定时任务自动生成摘要

### 推荐的 AI 服务

- **OpenAI**: [https://openai.com](https://openai.com)
- **DeepSeek**: [https://deepseek.com](https://deepseek.com)
- **其他兼容 OpenAI API 的服务**

## 环境变量（可选）

可以创建 `.env` 文件配置环境变量：

```bash
# Flask
FLASK_ENV=development
FLASK_DEBUG=1

# 数据库
DATABASE_URL=sqlite:///rss_reader.db

# AI 配置（可选，也可在界面配置）
AI_API_BASE_URL=https://api.openai.com/v1
AI_API_KEY=your-api-key
AI_MODEL=gpt-4o-mini
```

## 项目特色

- **无用户认证**: 简化部署，专注个人使用
- **AI 深度集成**: 智能总结、聚合分析、自动摘要
- **自动化**: 定时刷新、自动摘要、智能推荐
- **轻量级**: SQLite 数据库，零配置启动
- **现代化**: Nuxt 4 + Python 3.13，最新技术栈

---

## 📋 未来规划 (TODO)[]

### 🎙️ AI 播客功能 (优先级：高)[]

#### 阶段一：本地 TTS 集成[]

#### 阶段二：调侃式播客集成[]

#### 阶段三：小米小爱音箱对接[]

#### 阶段四：高级功能[]
---

## 许可证

GPLv3

本项目采用 GNU General Public License v3.0 开源许可证。详见 [LICENSE](LICENSE) 文件。

### 使用许可

- ✅ 商业使用
- ✅ 修改
- ✅ 分发
- ✅ 私人使用

### 条件和限制

- ⚠️ 必须包含原始许可证和版权声明
- ⚠️ 必须说明对文件的修改
- ⚠️ 必须以相同的许可证发布衍生作品
- ⚠️ 必须提供源代码
- ❌ 不得提供责任担保
- ❌ 不得使用作者名义进行宣传
