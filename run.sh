#!/bin/bash

# Export environment variables
export TELEGRAM_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
export TELEGRAM_ADMIN_ID="YOUR_ADMIN_ID_HERE"

# Webhook configuration
export WEBHOOK_POINT="https://your-domain.com/webhook"    # 替换为webhook接入点
export LISTEN_PORT="8443"                        # 注意，这里是程序监听的端口。如果程序位于Nginx/proxy后面，这个端口和WEBHOOK_POINT的端口可以不一致

# 如果您使用自签证书，取消注释并设置以下变量
# export CERT_FILE="/path/to/cert.pem"
# export KEY_FILE="/path/to/key.pem"

# Build and run the bot
go build -o tg-antispam
./tg-antispam