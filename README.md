# tg-antispam

Telegram 防止垃圾用户（主要是 Premium 用户）的 bot

## 功能特点

- 自动监控新加入群组的用户
- 识别并限制以下类型的可疑用户：
  - 会员用户(premium user)
  - 名称中含有 emoji 表情符号的用户
  - 用户名包含随机用户的用户
  - 被 [Combot Anti Spam](https://cas.chat) 标记的用户
- 对可疑用户自动限制发送消息和媒体的权限
- 向管理员发送封禁通知，对于错误封禁的用户可直接接触封禁

## 项目结构

项目采用标准的 Go 项目结构:

```
.
├── cmd/                  # 应用程序入口
│   └── tg-antispam/      # 主程序入口
├── configs/              # 配置文件
│   └── nginx/            # Nginx 配置示例
├── internal/             # 私有应用程序代码
│   ├── bot/              # Bot 相关代码
│   ├── handler/          # 消息处理器
│   ├── logger/           # 日志系统
│   └── models/           # 数据模型
├── scripts/              # 构建和运行脚本
│   ├── build.sh          # 构建脚本
│   └── run.sh            # 运行脚本
├── Dockerfile            # Docker 构建文件
├── docker-compose.yml    # Docker Compose 配置
├── go.mod                # Go 模块定义
└── README.md             # 项目说明
```

## 安装与使用

### 前置条件

- Go 1.24.1 或更高版本（直接构建运行）
- Docker 或 Docker Compose（Docker 部署）
- 一个 Telegram Bot Token（通过 [@BotFather](https://t.me/BotFather) 获取）
- 管理员的 Telegram 用户 ID（用于接收通知）
- 一个域名（需要能配置 https）

### 方式一：直接构建运行

1. 克隆仓库

```bash
git clone https://github.com/tlanyan/tg-antispam.git
cd tg-antispam
```

2. 构建项目

```bash
./scripts/build.sh
```

3. 设置环境变量

```bash
export TELEGRAM_BOT_TOKEN="your_token_here"
export TELEGRAM_ADMIN_ID="your_admin_id_here"
export WEBHOOK_POINT="https://your-domain.com/webhook"
export LISTEN_PORT="8443"
# 如果不使用Nginx/反向代理，需要设置证书文件路径
# export CERT_FILE="/path/to/cert.pem"
# export KEY_FILE="/path/to/key.pem"
```

4. 运行机器人

```bash
./build/tg-antispam
```

### 方式二：Docker 部署

1. 克隆仓库

```bash
git clone https://github.com/tlanyan/tg-antispam.git
cd tg-antispam
```

2. 设置环境变量

```bash
# 基本配置
echo "TELEGRAM_BOT_TOKEN=your_bot_token_here" > .env
echo "TELEGRAM_ADMIN_ID=your_admin_id_here" >> .env

# Webhook接入点配置
echo "WEBHOOK_POINT=https://your-domain.com/webhook" >> .env

echo "LISTEN_PORT=8443" >> .env

# 如果程序不经过Nginx/proxy, 直接监听和webhook_point中的端口，取消注释并设置SSL证书
# echo "CERT_FILE=/app/certs/cert.pem" >> .env
# echo "KEY_FILE=/app/certs/key.pem" >> .env
```

3. 使用 Docker Compose 构建并启动容器

```bash
docker-compose up -d
```

4. 查看日志

```bash
docker-compose logs -f
```

### Webhook 配置说明

Webhook 模式允许机器人实时接收消息更新，能更好地捕获被其他管理员快速删除的消息。配置要求：

1. **域名和 SSL 证书**：

   - 一个域名
   - SSL 证书，可复用现有业务的证书，也可通过 Let's Encrypt 获取免费证书。教程可参考 [使用 acme.sh 签发证书](https://itlanyan.com/use-acme-sh-get-free-cert/)

2. **端口要求**：

   - Telegram 只允许使用以下端口：443、80、88 或 8443
   - 默认配置使用 8443 端口

3. **环境变量配置**：

   - `WEBHOOK_POINT`: webhook 接入点，例如 "https://example.com/webhook"
   - `LISTEN_PORT`: 程序监听端口号，默认为 "8443"
   - `CERT_FILE`: SSL 证书文件路径（如果直接监听 webhook 回调地址端口）
   - `KEY_FILE`: SSL 密钥文件路径（如果直接监听 webhook 回调地址端口）

4. **使用 Nginx 等反向代理**：

   - 如果您已有服务器运行 Nginx 或 Apache，可以使用反向代理转发请求到该程序。
   - 此时无需设置 CERT_FILE 和 KEY_FILE，但 WEBHOOK_POINT 必须为 https://
   - Nginx 反向代理配置可参考 configs/nginx/server.conf

### 获取管理员 Telegram ID

向[@userinfobot](https://t.me/userinfobot)发送消息，它会返回你的用户 ID

### 添加到群组

1. 将机器人添加到 Telegram 群组
2. 将机器人设置为管理员，并授予以下权限：
   - 删除消息
   - 限制用户权限

## 注意事项

- 机器人需要拥有管理员权限才能限制用户
- 管理员需要先向机器人发送过私信，否则机器人无法向管理员发送私信通知
