# tg-antispam

Telegram 防止垃圾用户（主要是 Premium 用户）的 bot

## 功能特点

- 自动监控新加入群组的用户
- 识别并限制以下类型的可疑用户：
  - 会员用户(premium user)
  - 名称中含有 emoji 表情符号的用户
- 对可疑用户自动限制发送消息和媒体的权限
- 向管理员发送封禁通知（而非在群组内公开通知）
- 支持通过 Webhook 接收消息更新，提高消息捕获的完整性

## 安装与使用

### 前置条件

- Go 1.21 或更高版本（直接安装方式）
- Docker 和 Docker Compose（Docker 部署方式）
- 一个 Telegram Bot Token（通过 [@BotFather](https://t.me/BotFather) 获取）
- 管理员的 Telegram 用户 ID（用于接收通知）
- 对于 Webhook 模式：需要一个带 SSL 证书的域名或者公网 IP

### 方式一：直接安装运行

1. 克隆仓库

```bash
git clone https://github.com/tlanyan/tg-antispam.git
cd tg-antispam
```

2. 修改 `run.sh` 文件

   - 将 `YOUR_BOT_TOKEN_HERE` 替换为实际的 Bot Token
   - 将 `YOUR_ADMIN_ID_HERE` 替换为管理员的 Telegram 用户 ID
   - 对于 Webhook 模式，还需要设置 `WEBHOOK_POINT` 和 `LISTEN_PORT`

3. 运行机器人

```bash
./run.sh
```

### 方式二：使用 Docker 部署

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

# Webhook配置（如需使用Webhook）
echo "WEBHOOK_HOST=https://your-domain.com/webhook" >> .env
# 注意，这里是程序监听的端口。如果程序位于Nginx/proxy后面，这个端口和WEBHOOK_POINT的端口可以不一致
echo "LISTEN_PORT=8443" >> .env

# 如果使用自签证书，添加证书路径
# echo "CERT_FILE=/certs/cert.pem" >> .env
# echo "KEY_FILE=/certs/key.pem" >> .env
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

Webhook 模式允许机器人几乎实时接收消息更新，能更好地捕获被其他管理员快速删除的消息。配置要求：

1. **域名和 SSL 证书**：

   - 您需要一个具有有效 SSL 证书的公网域名
   - 可以使用 Let's Encrypt 获取免费的 SSL 证书

2. **端口要求**：

   - Telegram 只允许使用以下端口：443、80、88 或 8443
   - 默认配置使用 8443 端口

3. **环境变量配置**：

   - `WEBHOOK_HOST`: 您的域名，例如 "https://example.com"
   - `LISTEN_PORT`: 端口号，默认为 "8443"
   - `CERT_FILE`: SSL 证书文件路径（如需自签证书）
   - `KEY_FILE`: SSL 密钥文件路径（如需自签证书）

4. **使用反向代理**：
   - 如果您已有服务器运行 Nginx 或 Apache，可以使用反向代理转发请求到机器人
   - 此时无需设置 CERT_FILE 和 KEY_FILE，但 WEBHOOK_HOST 必须使用 https://

### 获取管理员 Telegram ID

可以通过以下方式获取：

1. 向 [@userinfobot](https://t.me/userinfobot) 发送消息，它会返回你的用户 ID
2. 或者使用 [@RawDataBot](https://t.me/RawDataBot)，它会提供更详细的信息

### 添加到群组

1. 将机器人添加到 Telegram 群组
2. 将机器人设置为管理员，并授予以下权限：
   - 删除消息
   - 限制用户权限

## 工作原理

1. 机器人监测新加入群组的成员
2. 当新成员加入时，机器人会检查其名称是否包含 emoji 以及用户名是否为无意义随机字符串
3. 如果符合上述条件之一，机器人会限制该用户的发言及媒体发送权限
4. 同时向管理员发送一条私人通知，说明限制原因和相关群组信息

## 注意事项

- 机器人需要拥有管理员权限才能限制用户
- 管理员需要先向机器人发送过私信，否则机器人无法向管理员发送私信通知
- 建议将机器人的优先级设置较高，以确保其能够在垃圾用户发送消息前进行限制

## 许可证

MIT
