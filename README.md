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

### 配置文件结构
复制 `config.example.json` 为 `config.json` 并根据需要修改配置项。

### Telegram Bot 配置
```json
"telegram": {
  "token": "your_bot_token_here",        // 从 @BotFather 获取的 Bot Token
  "mode": "getupdates",                  // 消息获取模式: getupdates 或 webhook
  "webhook_url": "",                     // webhook 模式下的回调URL
  "webhook_port": 8443,                  // webhook 监听端口
  "debug": true,                         // 是否显示调试信息
  "introduction": "..."                  // Bot 介绍信息
}
```

**消息获取模式说明：**
- `getupdates`: 主动拉取消息（推荐，适合大多数场景）
- `webhook`: 被动接收消息（需要公网域名和HTTPS）

### AI 聊天配置
```json
"chat": {
  "prefix": "",                          // 消息前缀，为空时所有消息触发AI
  "system_prompt": "...",                // 全局系统提示词
  "history_length": 5,                   // 对话历史保留条数 (0-50)
  "history_timeout_minutes": 30,         // 对话历史超时时间(分钟)
  "timeout": 60                          // 全局AI请求超时时间(秒)
}
```

### AI 提供商配置
**重要：建议只启用一个AI提供商避免冲突**

#### OpenAI 配置
```json
"openai": {
  "enabled": true,                       // 是否启用
  "api_key": "your_openai_api_key",      // API密钥
  "api_url": "https://api.openai.com/v1", // API端点
  "default_model": "gpt-3.5-turbo",     // 默认模型
  "disabled_models": [],                 // 禁用的模型列表
  "system_prompt": "",                   // 覆盖全局提示词
  "timeout": 0                           // 覆盖全局超时设置
}
```

**可用模型：** `gpt-3.5-turbo`, `gpt-4`, `gpt-4-turbo`, `gpt-4o`, `gpt-4o-mini`

#### Anthropic Claude 配置
```json
"anthropic": {
  "enabled": false,
  "api_key": "your_anthropic_api_key",
  "api_url": "https://api.anthropic.com",
  "default_model": "claude-3-sonnet-20240229",
  "disabled_models": [],
  "system_prompt": "",
  "timeout": 0
}
```

**可用模型：** `claude-3-haiku-20240307`, `claude-3-sonnet-20240229`, `claude-3-opus-20240229`, `claude-3-5-sonnet-20241022`

#### Google Gemini 配置
```json
"gemini": {
  "enabled": false,
  "api_key": "your_gemini_api_key",
  "api_url": "https://generativelanguage.googleapis.com/v1beta",
  "default_model": "gemini-pro",
  "disabled_models": [],
  "system_prompt": "",
  "timeout": 0
}
```

**可用模型：** `gemini-pro`, `gemini-pro-vision`, `gemini-1.5-pro`, `gemini-1.5-flash`

#### Ollama 配置（本地部署）
```json
"ollama": {
  "enabled": false,
  "api_url": "http://localhost:11434",
  "default_model": "llama2",
  "disabled_models": [],
  "system_prompt": "",
  "timeout": 0
}
```

**设置说明：**
1. 需要本地安装 Ollama: `https://ollama.ai/`
2. 下载模型: `ollama pull llama2`
3. 可用模型取决于已下载的模型

### 数据库配置
**重要：只能选择一种数据库类型**

```json
"database": {
  "type": "json"                         // 数据库类型: json, sqlite, mysql, postgresql
}
```

**类型选择建议：**
- **开发环境**: `json` 或 `sqlite`
- **生产环境**: `mysql` 或 `postgresql`

#### JSON 文件数据库（默认）
```json
"database": {
  "type": "json",
  "json": {
    "filename": "data.json"              // 数据文件路径
  }
}
```
**优点**: 轻量级，无需额外安装  
**缺点**: 不支持并发，适合小规模使用

#### SQLite 数据库
```json
"database": {
  "type": "sqlite",
  "sqlite": {
    "filename": "bot_data.db"            // 数据库文件路径
  }
}
```
**优点**: 轻量级，支持SQL，文件存储  
**缺点**: 并发能力有限

#### MySQL 数据库
```json
"database": {
  "type": "mysql",
  "mysql": {
    "host": "localhost",                 // 数据库主机
    "port": 3306,                        // 端口号
    "user": "bot_user",                  // 用户名
    "password": "your_mysql_password",   // 密码
    "database": "telegram_bot",          // 数据库名
    "sslmode": "false"                   // SSL模式
  }
}
```

**MySQL SSL模式选项：**
- `false`: 禁用SSL（默认）
- `true`: 启用SSL，但不验证证书
- `skip-verify`: 启用SSL，跳过证书验证
- `preferred`: 优先使用SSL，失败时回退到非SSL
- `disable`: 强制禁用SSL

#### PostgreSQL 数据库
```json
"database": {
  "type": "postgresql",
  "postgresql": {
    "host": "localhost",                 // 数据库主机
    "port": 5432,                        // 端口号
    "user": "bot_user",                  // 用户名
    "password": "your_postgresql_password", // 密码
    "database": "telegram_bot",          // 数据库名
    "sslmode": "disable"                 // SSL模式
  }
}
```

**PostgreSQL SSL模式选项：**
- `disable`: 禁用SSL
- `require`: 要求SSL连接
- `verify-ca`: 验证CA证书
- `verify-full`: 完全验证证书

### Redis 缓存配置（可选）
```json
"redis": {
  "enabled": false,                      // 是否启用Redis缓存
  "host": "localhost",                   // Redis主机
  "port": 6379,                          // 端口号
  "password": "",                        // 密码（无密码时留空）
  "database": 0,                         // 数据库编号 (0-15)
  "ttl": 1800,                           // 对话缓存过期时间(秒)
  "ai_cache_enabled": false,             // 是否启用AI对话缓存
  "ai_cache_ttl": 3600                   // AI对话缓存过期时间(秒)
}
```

**Redis缓存功能：**
- **对话缓存**: 存储用户对话历史，提高响应速度
- **AI缓存**: 缓存AI回复，相同问题直接返回缓存结果

**AI缓存说明：**
- 按照 `渠道->模型->问题` 的组合进行缓存
- 命中缓存时直接回复缓存内容，并显示"💾 缓存回复"
- 缓存回复不计入对话轮数，不显示tokens统计
- 缓存回复不会被加入对话上下文

**建议：** 生产环境启用Redis提高性能

### 管理员权限配置
```json
"admin": {
  "super_admin_ids": [123456789],        // 超级管理员ID列表
  "admin_ids": [],                       // 普通管理员ID列表
  "allowed_group_ids": []                // 允许使用的群组ID列表
}
```

**获取用户ID：** 发送 `/start` 给bot查看自己的用户ID  
**群组白名单：** 空数组表示允许所有群组

### 环境变量配置（推荐）
为了提高安全性，建议使用环境变量存储敏感信息：

```bash
# Windows PowerShell
$env:TELEGRAM_BOT_TOKEN="your_bot_token"
$env:OPENAI_API_KEY="your_openai_key"
$env:ANTHROPIC_API_KEY="your_anthropic_key"
$env:GEMINI_API_KEY="your_gemini_key"

# Linux/macOS
export TELEGRAM_BOT_TOKEN="your_bot_token"
export OPENAI_API_KEY="your_openai_key"
export ANTHROPIC_API_KEY="your_anthropic_key"
export GEMINI_API_KEY="your_gemini_key"
```

**优先级：** 环境变量 > 配置文件

### 部署模式建议

#### 开发环境
```json
{
  "telegram": { "mode": "getupdates", "debug": true },
  "database": { "type": "json" },
  "redis": { "enabled": false }
}
```

#### 生产环境
```json
{
  "telegram": { "mode": "webhook", "debug": false },
  "database": { "type": "mysql" },
  "redis": { "enabled": true }
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

<!--GAMFC_DELIMITER-->
<!--GAMFC_DELIMITER_END-->