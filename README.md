# TGFaqBot 使用指南

## 🚀 快速开始

### 1. 环境配置

#### 设置环境变量（推荐）
```bash
# Windows PowerShell
$env:TELEGRAM_BOT_TOKEN="your_bot_token_here"
$env:OPENAI_API_KEY="your_openai_api_key"

# Linux/macOS
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
export OPENAI_API_KEY="your_openai_api_key"
```

#### 配置文件
如果不使用环境变量，可以在 `config.json` 中设置（会显示安全警告）：
```json
{
  "telegram": {
    "token": "your_bot_token_here",
    "mode": "getupdates"
  },
  "chat": {
    "openai": {
      "enabled": true,
      "api_key": "your_openai_api_key"
    }
  }
}
```

### 2. 运行Bot

```bash
# 编译
go build -o TGFaqBot.exe .

# 运行
./TGFaqBot.exe
```

## 🤖 功能介绍

### AI对话功能
- **智能回复**: 当启用AI提供商时，bot会智能回复用户消息
- **静默模式**: 当没有启用AI提供商时，bot将静默处理消息，不会回复
- **多提供商支持**: 支持OpenAI、Anthropic、Gemini、Ollama等多种AI服务

### 用户命令
- `/start` - 显示介绍信息
- `/query <关键词>` - 查询FAQ内容
- `/commands` - 显示可用命令
- `/userinfo` - 显示用户信息
- `/models` - 显示可用AI模型
- `/clearchat` - 清除对话历史

### 管理员命令
- `/add` - 添加FAQ条目
- `/update` - 更新FAQ条目
- `/delete` - 删除FAQ条目
- `/batchdelete` - 批量删除FAQ条目
- `/list` - 列出所有条目
- `/reload` - 重新加载数据库
- `/history` - 查看操作历史
- `/undo` - 撤销最近的操作

### 超级管理员命令
- `/addadmin` - 添加管理员
- `/deladmin` - 删除管理员
- `/addgroup` - 添加允许的群组
- `/delgroup` - 删除允许的群组

## 🔧 配置说明

### 数据库配置
支持四种数据库类型：

#### JSON文件（默认）
```json
"database": {
  "type": "json",
  "json": {
    "filename": "database.json"
  }
}
```

#### SQLite
```json
"database": {
  "type": "sqlite",
  "sqlite": {
    "filename": "database.db"
  }
}
```

#### MySQL
```json
"database": {
  "type": "mysql",
  "mysql": {
    "host": "localhost",
    "port": 3306,
    "user": "username",
    "password": "password",
    "database": "botdb",
    "sslmode": "false"
  }
}
```

**MySQL SSL模式说明**：
- `false`: 禁用SSL（默认）
- `true`: 启用SSL，但不验证证书
- `skip-verify`: 启用SSL，跳过证书验证
- `preferred`: 优先使用SSL，失败时回退到非SSL
- `disable`: 强制禁用SSL

#### PostgreSQL
```json
"database": {
  "type": "postgresql",
  "postgresql": {
    "host": "localhost",
    "port": 5432,
    "user": "username",
    "password": "password",
    "database": "botdb",
    "sslmode": "disable"
  }
}
```

### AI提供商配置

#### OpenAI
```json
"openai": {
  "enabled": true,
  "api_key": "sk-...",
  "api_url": "https://api.openai.com/v1",
  "default_model": "gpt-4o-mini"
}
```

#### Anthropic Claude
```json
"anthropic": {
  "enabled": true,
  "api_key": "sk-ant-...",
  "api_url": "https://api.anthropic.com",
  "default_model": "claude-3-haiku-20240307"
}
```

#### Google Gemini
```json
"gemini": {
  "enabled": true,
  "api_key": "AIza...",
  "api_url": "https://generativelanguage.googleapis.com/v1beta",
  "default_model": "gemini-pro"
}
```

## 🔒 安全特性

1. **环境变量支持**: 敏感信息可通过环境变量配置
2. **配置验证**: 启动时验证配置完整性
3. **速率限制**: 防止用户滥用（10次/分钟）
4. **权限管理**: 分级管理员权限
5. **群组白名单**: 只在允许的群组中工作

## 🧪 开发和测试

### 运行测试
```bash
# 运行所有测试
go test ./... -v

# 运行特定模块测试
go test ./database -v
go test ./utils -v
```

### 代码格式化
```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...
```

## 📝 FAQ匹配类型

1. **精确匹配** (type=1): 完全匹配关键词
2. **包含匹配** (type=2): 包含关键词的内容
3. **正则匹配** (type=3): 正则表达式模式匹配

## 🔍 故障排除

### 常见问题

#### Bot无法启动
- 检查Telegram Token是否正确
- 确认网络连接正常
- 查看日志中的错误信息

#### 无法连接AI服务
- 验证API密钥是否有效
- 检查API URL是否正确
- 确认账户有足够余额

#### PostgreSQL连接问题
- 确认PostgreSQL服务正在运行
- 检查连接参数（主机、端口、用户名、密码）
- 验证数据库是否存在
- 检查SSL模式设置（disable/require/verify-ca/verify-full）
- 确认用户具有数据库访问权限

#### MySQL连接问题
- 确认MySQL服务正在运行
- 检查连接参数（主机、端口、用户名、密码）
- 验证数据库是否存在
- 检查SSL模式设置（false/true/skip-verify/preferred）
- 确认用户具有数据库访问权限

#### 数据库错误
- 确认数据库文件权限正确
- 检查数据库连接参数
- 验证数据库结构完整性

#### 交互操作超时或中断
- 交互操作有时间限制，长时间无响应会自动取消
- 可以随时发送 `/cancel` 或点击"取消"按钮中断操作
- 如果操作被意外中断，可以重新开始流程

### 日志级别
设置 `debug: true` 可查看详细日志：
```json
"telegram": {
  "debug": true
}
```

## 💡 交互流程优化建议

### 已实现的优化功能

#### 1. **操作超时机制** ✅
- 多级交互操作设置5分钟超时限制
- 超时后自动清理状态并通知用户
- 每2分钟自动检查和清理过期会话

#### 2. **批量操作支持** ✅
- 新增 `/batchdelete` 命令支持批量删除FAQ条目
- 支持按匹配类型和关键词模式筛选
- 提供删除预览和确认机制

#### 3. **预览和确认机制** ✅
- 更新操作前显示当前内容预览
- 删除操作增加二次确认界面
- 批量删除显示影响条目数量和预览

#### 4. **操作历史和撤销** ✅
- 记录最近10次操作的详细历史
- 支持撤销5分钟内的单个操作（添加、更新、删除）
- 提供操作历史查看功能

#### 5. **改进的取消机制** ✅
- 取消操作后提供返回主菜单选项
- 清理相关的对话状态
- 更友好的用户提示

### 当前多级交互流程
项目支持以下复杂的多级交互场景：

1. **FAQ条目管理流程**:
   ```
   /list → 选择条目 → 选择操作 → 输入参数 → 预览确认 → 执行
   ```

2. **批量删除流程**:
   ```
   /batchdelete → 筛选条件 → 预览条目 → 确认删除 → 执行
   ```

3. **管理员管理流程**:
   ```
   /listadmin → 选择管理员 → 确认操作
   ```

## 开发者

<!--GAMFC_DELIMITER--><a href="https://github.com/HsukqiLee" title="Hsukqi Lee"><img src="https://avatars.githubusercontent.com/u/79034142?v=4" width="50;" alt="Hsukqi Lee"/></a><!--GAMFC_DELIMITER_END-->