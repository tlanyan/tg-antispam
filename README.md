# tg-antispam

Telegram防止垃圾用户的bot

## 功能特点

- 自动监控新加入群组的用户
- 识别并限制以下类型的可疑用户：
  - 名称中含有emoji表情符号的用户
  - 用户名是无意义随机字符串的用户
- 对可疑用户自动限制发送消息和媒体的权限
- 向管理员发送封禁通知（而非在群组内公开通知）

## 安装与使用

### 前置条件

- Go 1.21 或更高版本（直接安装方式）
- Docker 和 Docker Compose（Docker 部署方式）
- 一个 Telegram Bot Token（通过 [@BotFather](https://t.me/BotFather) 获取）
- 管理员的 Telegram 用户 ID（用于接收通知）

### 方式一：直接安装运行

1. 克隆仓库
```bash
git clone https://github.com/tlanyan/tg-antispam.git
cd tg-antispam
```

2. 修改 `run.sh` 文件
   - 将 `YOUR_BOT_TOKEN_HERE` 替换为实际的 Bot Token
   - 将 `YOUR_ADMIN_ID_HERE` 替换为管理员的 Telegram 用户 ID

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
echo "TELEGRAM_BOT_TOKEN=your_bot_token_here" > .env
echo "TELEGRAM_ADMIN_ID=your_admin_id_here" >> .env
```

3. 使用 Docker Compose 构建并启动容器
```bash
docker-compose up -d
```

4. 查看日志
```bash
docker-compose logs -f
```

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