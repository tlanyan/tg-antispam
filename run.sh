#!/bin/bash

# Export environment variables
export TELEGRAM_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
export TELEGRAM_ADMIN_ID="YOUR_ADMIN_ID_HERE"

# Build and run the bot
go build -o tg-antispam
./tg-antispam 