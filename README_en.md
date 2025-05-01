# tg-antispam

A Telegram bot for preventing spam users (primarily Premium users).

[中文版](./README.md)

## Features

- Automatically monitors users joining groups
- Identifies and restricts suspicious users:
  - Premium users
  - Users with emoji in their names
  - Users with random usernames
  - Users flagged by [Combot Anti Spam](https://cas.chat)
- Automatically restricts message sending and media permissions for suspicious users
- Sends ban notifications to administrators, with options to unban users in case of false positives

## Project Structure

The project follows a standard Go project structure:

```
.
├── cmd/                  # Application entrypoints
│   └── tg-antispam/      # Main program entrypoint
├── configs/              # Configuration files
│   └── nginx/            # Nginx configuration examples
│   └── config.yaml       # YAML configuration file
├── internal/             # Private application code
│   ├── bot/              # Bot-related code
│   ├── config/           # Configuration system
│   ├── handler/          # Message handlers
│   ├── logger/           # Logging system
│   └── models/           # Data models
├── scripts/              # Build and run scripts
│   ├── build.sh          # Build script
│   └── run.sh            # Run script
├── Dockerfile            # Docker build file
├── docker-compose.yml    # Docker Compose configuration
├── go.mod                # Go module definition
└── README.md             # Project documentation
```

## Configuration System

TG-Antispam uses a YAML configuration file for all settings. The default configuration file is located at `config.yaml`. You can specify a different configuration file location using the `-config` command line parameter:

```bash
./tg-antispam -config=/path/to/config.yaml
```

## Installation and Usage

If you don't want to deploy the bot yourself, you can add [@justgodiebot](https://t.me/justgodiebot) to your group and set it as an administrator. If you want to receive ban notifications, you need to interact with the bot once.

### Prerequisites

- Go 1.24.1 or higher (for direct build and run)
- Docker or Docker Compose (for Docker deployment)
- A Telegram Bot Token (obtain from [@BotFather](https://t.me/BotFather))
- A domain name (with HTTPS capability)

### Method 1: Direct Build and Run

1. Clone the repository

```bash
git clone https://github.com/tlanyan/tg-antispam.git
cd tg-antispam
```

2. Build the project

```bash
./scripts/build.sh
```

3. Configure

Edit `configs/config.yaml` and fill in the necessary information

4. Run the bot

```bash
./build/tg-antispam -config=/path/to/config.yaml
```

### Method 2: Docker Deployment

1. Clone the repository

```bash
git clone https://github.com/tlanyan/tg-antispam.git
cd tg-antispam
```

2. Configure

Edit the `configs/config.yaml` file and fill in the necessary configuration information

3. Build and start the container using Docker Compose

```bash
docker-compose up -d
```

4. View logs

```bash
docker-compose logs -f
```

### Webhook Configuration

Webhook mode allows the bot to receive message updates in real-time, which helps better capture messages that might be quickly deleted by other administrators. Configuration requirements:

1. **Domain and SSL Certificate**:

   - A domain name
   - SSL certificate, which can reuse existing business certificates or obtain free certificates through Let's Encrypt. For a tutorial, see [Using acme.sh to Issue Certificates](https://itlanyan.com/use-acme-sh-get-free-cert/)

2. **Port Requirements**:

   - Telegram only allows the following ports: 443, 80, 88, or 8443
   - Default configuration uses port 8443

3. **Configuration Options**:

   - `bot.webhook.endpoint`: webhook endpoint, e.g., "https://example.com/webhook"
   - `bot.webhook.port`: port the program listens on, default is "8443"
   - `bot.webhook.cert_file`: SSL certificate file path (if directly listening on the webhook callback address port)
   - `bot.webhook.key_file`: SSL key file path (if directly listening on the webhook callback address port)

4. **Using Nginx or Other Reverse Proxies**:

   - If you already have a server running Nginx or Apache, you can use a reverse proxy to forward requests to this program
   - In this case, you don't need to set cert_file and key_file, but webhook.endpoint must be https://
   - For Nginx reverse proxy configuration, refer to `configs/nginx/server.conf`

### Adding to a Group

1. Add the bot to your Telegram group
2. Set the bot as an administrator and grant the following permissions:

   - Delete messages
   - Ban users

3. If you want to receive notifications from the bot, you need to interact with it once

## Notes

- The bot needs administrator privileges to restrict users
- Administrators need to have sent a private message to the bot first, otherwise the bot cannot send private notification messages to administrators
