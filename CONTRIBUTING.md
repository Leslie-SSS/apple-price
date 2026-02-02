# Contributing to Apple Price

感谢您对 Apple Price 项目的关注！我们欢迎任何形式的贡献。

## 开发环境设置

### 前置要求
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose

### 本地开发

```bash
# 启动开发环境
docker compose up -d

# 前端开发
cd frontend && npm install && npm run dev

# 后端开发
cd backend && go run cmd/server/main.go
```

## 贡献流程

### 1. Fork 仓库
点击右上角 Fork 按钮创建您的副本

### 2. 创建分支
```bash
git checkout -b feat/your-feature-name
# 或
git checkout -b fix/your-bug-fix
```

### 3. 提交代码
```bash
git add .
git commit -m "feat: add some feature"
```

### 4. 推送到分支
```bash
git push origin feat/your-feature-name
```

### 5. 创建 Pull Request
访问 GitHub 页面创建 PR

## 代码规范

### 提交信息规范
遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

- `feat:` 新功能
- `fix:` Bug 修复
- `docs:` 文档更新
- `style:` 代码格式调整
- `refactor:` 重构
- `test:` 测试相关
- `chore:` 构建/工具相关

### 代码风格

**Go (Backend):**
- 使用 `gofmt` 格式化代码
- 遵循 [Effective Go](https://go.dev/doc/effective_go) 指南

**TypeScript (Frontend):**
- 使用 ESLint + Prettier
- 遵循 React Hooks 规则

## Pull Request 检查清单

提交 PR 前请确认：

- [ ] 代码通过所有测试
- [ ] 代码符合项目风格规范
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
- [ ] 提交信息清晰描述变更内容

## 报告问题

报告 Bug 时请提供：

- 环境信息（操作系统、版本）
- 复现步骤
- 预期行为
- 实际行为
- 截图或日志

## 行为准则

- 尊重他人
- 接受建设性批评
- 关注对社区最有利的事情

## 获取帮助

- [GitHub Issues](../../issues) - 报告问题
- [GitHub Discussions](../../discussions) - 提出问题

---

再次感谢您的贡献！
