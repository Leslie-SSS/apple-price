# ApplePrice - 苹果官方翻新产品价格监听工具

监听苹果中国大陆和香港地区官方翻新产品，提供智能分析和价格追踪。

## 功能特性

- 🔄 **自动爬取**: 每5分钟自动爬取 Apple CN/HK 翻新产品
- 📊 **价格追踪**: 记录价格历史，展示价格变动趋势
- 🔔 **价格通知**: 支持 Bark 推送通知（iOS）
- 🔔 **上新通知**: 新品上架自动通知
- 💾 **数据持久化**: SQLite 数据库存储

## 技术栈

- **后端**: Go (Gin框架)
- **前端**: React + Vite + TailwindCSS
- **存储**: SQLite 数据库
- **通知**: Bark API

## 快速开始

### 本地开发

#### 后端

```bash
cd backend

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，设置必要的配置

# 运行服务
go run cmd/server/main.go
```

#### 前端

```bash
cd frontend

# 安装依赖
npm install

# 运行开发服务器
npm run dev

# 构建生产版本
npm run build
```

### Docker 部署

```bash
# 复制环境变量文件
cp .env.example .env

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

## API 文档

### 产品接口

- `GET /api/products` - 获取产品列表
  - Query: `category`, `region`, `sort`, `order`, `stock_status`
- `GET /api/products/:id` - 获取产品详情
- `GET /api/products/:id/history` - 获取价格历史

### 订阅接口

- `POST /api/subscriptions` - 创建价格订阅
- `DELETE /api/subscriptions/:id` - 删除订阅
- `GET /api/subscriptions` - 获取订阅列表

### 新品订阅接口

- `POST /api/new-arrival-subscriptions` - 创建新品订阅
- `DELETE /api/new-arrival-subscriptions/:id` - 删除新品订阅
- `GET /api/new-arrival-subscriptions` - 获取新品订阅列表

### 其他接口

- `GET /api/categories` - 获取分类列表
- `GET /api/stats` - 获取统计信息
- `GET /api/health` - 健康检查
- `POST /api/scrape` - 手动触发爬取

## 目录结构

```
apple-price/
├── backend/                 # Go 后端
│   ├── cmd/server/         # 主程序入口
│   ├── internal/
│   │   ├── api/           # HTTP handlers
│   │   ├── scraper/       # 翻新产品爬虫
│   │   ├── notify/        # Bark 通知服务
│   │   ├── store/         # SQLite 数据库
│   │   ├── model/         # 数据模型
│   │   └── config/        # 配置管理
│   └── data/              # 数据存储目录
├── frontend/              # React 前端
│   ├── src/
│   │   ├── components/   # 组件
│   │   ├── pages/        # 页面
│   │   ├── hooks/        # 自定义 hooks
│   │   ├── services/     # API 调用
│   │   └── utils/        # 工具函数
│   └── public/
├── config/               # 配置文件
└── docker-compose.yml    # 容器编排
```

## Bark 推送通知配置

本项目使用 [Bark](https://github.com/Finb/Bark) 作为 iOS 推送通知服务。Bark 是一款开源的自定义推送工具，支持通过 API 发送通知到 iOS 设备。

### 获取 Bark Key

1. **下载 Bark App**
   - 在 App Store 搜索 "Bark" 并下载
   - 或访问官网: https://github.com/Finb/Bark

2. **获取推送 Key**
   - 打开 Bark App
   - 首页会显示你的推送 Key（类似：`xxxxx` 的一串字符）
   - 点击复制即可

3. **配置订阅**
   - 在产品页面点击"订阅通知"
   - 粘贴你的 Bark Key
   - 可选：设置目标价格，低于该价格时才会通知

### 新品订阅

新品订阅允许你订阅特定类型的产品上架通知：

- **分类筛选**: 选择 Mac、iPad、iPhone、Watch 等
- **价格区间**: 设置最低/最高价格范围
- **关键词**: 产品名称包含指定关键词时通知

### 通知类型

| 通知类型 | 触发条件 |
|---------|---------|
| 价格变动 | 订阅的产品价格发生变化 |
| 目标价提醒 | 产品价格降至目标价以下 |
| 新品上架 | 符合条件的新产品上架 |

## 配置说明

### 后端环境变量

| 变量 | 说明 | 默认值 |
|-----|------|-------|
| `ENVIRONMENT` | 环境 | `development` |
| `HOST` | 监听地址 | `0.0.0.0` |
| `PORT` | 端口 | `8080` |
| `CORS_ORIGINS` | CORS 允许源 | `*` |
| `SCRAPER_INTERVAL` | 爬取间隔 | `5m` |
| `SCRAPER_USER_AGENT` | 爬虫 UA | 默认值 |
| `DATA_DIR` | 数据目录 | `./data` |

## 致谢

- [Bark](https://github.com/Finb/Bark) - 优秀的 iOS 自定义推送通知工具，本项目使用 Bark 作为推送通知服务

## 许可证

MIT
