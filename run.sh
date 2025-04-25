#!/bin/bash

# Export environment variables
export TELEGRAM_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
export TELEGRAM_ADMIN_ID="YOUR_ADMIN_ID_HERE"

# Webhook configuration
export WEBHOOK_HOST="https://your-domain.com"    # 替换为您的域名
export WEBHOOK_PATH="/webhook"                    # webhook路径
export WEBHOOK_PORT="8443"                        # webhook端口

# 如果您使用自签证书，取消注释并设置以下变量
# export CERT_FILE="/path/to/cert.pem"
# export KEY_FILE="/path/to/key.pem"

# Build and run the bot
go build -o tg-antispam
./tg-antispam