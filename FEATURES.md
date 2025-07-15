# TGFaqBot 新功能说明

## 最新更新 (v2.0)

### 🔄 重试功能 (/retry)
- **新增命令**: `/retry` - 重新生成上一次AI回复
- **功能特点**: 
  - 保持对话上下文，只重新生成最后一条AI回复
  - 旧的响应消息不会被删除，但不会加入上下文历史
  - 重试也算作一次对话，会计入对话上限
  - 支持流式输出和实时更新
  - 自动使用用户偏好的AI模型

### 📦 Redis缓存支持
- **可选配置**: Redis缓存提升性能
- **功能优势**:
  - 对话数据持久化存储
  - 跨重启保持对话状态
  - 提高响应速度
  - 支持分布式部署

### 🔄 智能模型缓存机制
- **新增功能**: 模型列表智能缓存策略
- **工作原理**:
  - 启动时优先尝试从API获取最新模型列表
  - 如果API获取失败，检查数据库中的缓存模型
  - 如果缓存模型更新时间在24小时内，使用缓存模型
  - 如果缓存过期或不存在，则使用内置默认模型
- **优势**: 提高启动稳定性，减少因网络问题导致的模型列表为空

#### Redis配置示例：
```json
{
  "redis": {
    "enabled": true,
    "host": "localhost",
    "port": 6379,
    "password": "your_redis_password",
    "database": 0,
    "ttl": 1800
  }
}
```

- `enabled`: 是否启用Redis (true/false)
- `host`: Redis服务器地址
- `port`: Redis端口 (默认6379)
- `password`: Redis密码 (可选)
- `database`: Redis数据库编号 (默认0)
- `ttl`: 数据TTL时间(秒) (默认1800=30分钟)

### 🎨 Markdown渲染优化
- **改进**: 使用 `github.com/zavitkov/tg-markdown` 库
- **功能**: 标准Markdown转换为Telegram MarkdownV2格式
- **效果**: AI回复中的格式化文本正确显示
- **回退**: 自动降级到普通文本避免解析错误

### 📋 使用说明

#### 基本对话流程：
1. 发送消息与AI对话
2. 如需重新生成回复，使用 `/retry`
3. 使用 `/models` 选择偏好的AI模型
4. 使用 `/clearchat` 清除对话历史

#### 重试功能示例：
```
用户: 帮我写一首诗
AI: [生成的诗歌]
用户: /retry
AI: [重新生成另一首诗]
```

#### Redis部署提示：
- 开发环境: 可不启用Redis，使用内存存储
- 生产环境: 建议启用Redis提升性能
- 集群部署: 必须启用Redis实现数据共享

### 🔧 环境变量支持
现支持以下环境变量：
- `TELEGRAM_BOT_TOKEN`: Telegram Bot Token
- `OPENAI_API_KEY`: OpenAI API密钥
- `ANTHROPIC_API_KEY`: Anthropic API密钥
- `GEMINI_API_KEY`: Google Gemini API密钥

### 📊 性能优化
- **流式响应**: 真正的流式输出，实时更新消息
- **智能防抖**: 优化消息更新频率
- **Token计数**: 准确统计输入输出token数量
- **错误处理**: 渐进式格式降级避免消息发送失败

### 🏗️ 技术架构
- **多Provider支持**: OpenAI、Anthropic、Gemini、Ollama
- **模型偏好系统**: 每个聊天会话独立的模型选择
- **Redis集成**: 可选的高性能缓存层  
- **流式处理**: SSE流式解析和实时消息更新
- **Markdown转换**: 智能格式化渲染

## 升级说明

从v1.x升级到v2.0：
1. 更新配置文件，添加Redis配置节点（如需要）
2. 重新编译：`go build -o TGFaqBot.exe .`
3. 重启Bot服务
4. 享受新功能！

配置文件向后兼容，无需修改现有配置即可使用。
