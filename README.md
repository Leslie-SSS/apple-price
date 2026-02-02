# ApplePrice - 苹果官方翻新产品价格监听工具

监听苹果中国大陆和香港地区官方翻新产品，提供AI分析总结和推荐值。

## 功能特性

- 🔄 **自动爬取**: 每5分钟自动爬取 Apple CN/HK 翻新产品
- 🤖 **AI 分析**: 基于 DeepSeek API 生成推荐值和标签
- 📊 **价格追踪**: 记录价格历史，展示价格变动趋势
- 🔔 **价格通知**: 支持 Bark 和 Email 通知
- 💾 **本地订阅**: 用户订阅信息存储在浏览器本地

## 技术栈

- **后端**: Go (Gin框架)
- **前端**: React + Vite + TailwindCSS
- **存储**: 内存 + JSON文件持久化
- **AI**: DeepSeek API
- **通知**: Bark 公共API + SMTP

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

- `POST /api/subscriptions` - 创建订阅
- `DELETE /api/subscriptions/:id` - 删除订阅
- `GET /api/subscriptions` - 获取订阅列表

### 其他接口

- `GET /api/categories` - 获取分类列表
- `GET /api/stats` - 获取统计信息
- `GET /api/health` - 健康检查

## 目录结构

```
apple-price/
├── backend/                 # Go 后端
│   ├── cmd/server/         # 主程序入口
│   ├── internal/
│   │   ├── api/           # HTTP handlers
│   │   ├── scraper/       # 翻新产品爬虫
│   │   ├── ai/            # DeepSeek AI 集成
│   │   ├── notify/        # Bark + Email 通知
│   │   ├── store/         # 内存存储 + JSON持久化
│   │   ├── model/         # 数据模型
│   │   └── config/        # 配置管理
│   └── data/              # JSON数据持久化目录
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

## 配置说明

### 后端环境变量

| 变量 | 说明 | 默认值 |
|-----|------|-------|
| `ENVIRONMENT` | 环境 | `development` |
| `PORT` | 端口 | `8080` |
| `DEEPSEEK_API_KEY` | DeepSeek API Key | - |
| `SMTP_HOST` | SMTP 服务器 | `smtp.gmail.com` |
| `SMTP_PORT` | SMTP 端口 | `587` |
| `SMTP_USER` | SMTP 用户名 | - |
| `SMTP_PASSWORD` | SMTP 密码 | - |
| `SCRAPER_INTERVAL` | 爬取间隔 | `5m` |

### Bark 配置

用户需在 App Store 下载 Bark App，获取 Bark Key 后在前端设置。

## 许可证

MIT
