package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

var (
	// Compiled regular expressions
	emojiRegex = regexp.MustCompile(`[\x{1F600}-\x{1F64F}|\x{1F300}-\x{1F5FF}|\x{1F680}-\x{1F6FF}|\x{1F700}-\x{1F77F}|\x{1F780}-\x{1F7FF}|\x{1F800}-\x{1F8FF}|\x{1F900}-\x{1F9FF}|\x{1FA00}-\x{1FA6F}|\x{1FA70}-\x{1FAFF}|\x{2600}-\x{26FF}|\x{2700}-\x{27BF}]`)
)

func main() {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	// Get webhook configuration from environment variables
	webhookPoint := os.Getenv("WEBHOOK_POINT")
	if webhookPoint == "" {
		log.Fatal("WEBHOOK_POINT environment variable not set (e.g. https://example.com/webhook)")
	}

	// 解析URL以获取路径部分
	parsedURL, err := url.Parse(webhookPoint)
	if err != nil {
		log.Fatalf("Invalid WEBHOOK_POINT: %v", err)
	}
	webhookPath := parsedURL.Path
	if webhookPath == "" {
		webhookPath = "/webhook"
		log.Printf("No path specified in WEBHOOK_POINT, using default path: %s", webhookPath)
	}

	listenPort := os.Getenv("LISTEN_PORT")
	if listenPort == "" {
		listenPort = "8443" // Default listen port
		log.Printf("Using default listen port: %s", listenPort)
	}

	webhookListen := "0.0.0.0:" + listenPort
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")

	if (certFile == "" || keyFile == "") && !strings.HasPrefix(webhookPoint, "https://") {
		log.Fatal("HTTPS configuration required: Set CERT_FILE and KEY_FILE env vars or use a HTTPS proxy")
	}

	// Initialize bot
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Get bot info
	botUser, err := bot.GetMe(ctx)
	if err != nil {
		log.Fatalf("Failed to get bot info: %v", err)
	}
	log.Printf("Authorized on account %s", botUser.Username)

	// Delete any existing webhook
	err = bot.DeleteWebhook(ctx, &telego.DeleteWebhookParams{})
	if err != nil {
		log.Fatalf("Failed to delete existing webhook: %v", err)
	}

	// 设置固定的secret token，而不是随机生成
	secretToken := "secure_webhook_token_" + botToken[len(botToken)-6:]

	// Set up webhook
	log.Printf("Setting webhook to: %s", webhookPoint)
	setWebhookParams := &telego.SetWebhookParams{
		URL: webhookPoint,
		// 指定需要接收的更新类型
		AllowedUpdates: []string{"message", "channel_post", "chat_member", "my_chat_member"},
		SecretToken:    secretToken,
	}

	err = bot.SetWebhook(ctx, setWebhookParams)
	if err != nil {
		log.Fatalf("Failed to set webhook: %v", err)
	}

	// 获取并显示webhook信息以进行调试
	webhookInfo, err := bot.GetWebhookInfo(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get webhook info: %v", err)
	} else {
		log.Printf("Webhook info: URL=%s, HasCustomCert=%v, PendingUpdateCount=%d",
			webhookInfo.URL, webhookInfo.HasCustomCertificate, webhookInfo.PendingUpdateCount)
		if webhookInfo.LastErrorDate > 0 {
			log.Printf("Webhook last error: [%d] %s", webhookInfo.LastErrorDate, webhookInfo.LastErrorMessage)
		}
		log.Printf("Allowed updates: %v", webhookInfo.AllowedUpdates)
	}

	// Create HTTP server mux
	mux := http.NewServeMux()

	// 创建一个同步等待组，用于确保服务器已准备好接收请求
	var serverReady sync.WaitGroup
	serverReady.Add(1)

	// 添加一个调试handler，记录所有收到的HTTP请求
	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Debug endpoint accessed: %s %s", r.Method, r.URL.Path)

		// 显示请求头和内容
		log.Printf("Request headers: %v", r.Header)

		// 返回更详细的状态信息
		webhookInfo, err := bot.GetWebhookInfo(ctx)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		response := "Bot webhook server is running\n\n"
		response += fmt.Sprintf("Bot username: %s\n", botUser.Username)
		response += fmt.Sprintf("Webhook path: %s\n", webhookPath)

		if err == nil {
			response += fmt.Sprintf("\nWebhook Info:\n")
			response += fmt.Sprintf("URL: %s\n", webhookInfo.URL)
			response += fmt.Sprintf("Custom Certificate: %v\n", webhookInfo.HasCustomCertificate)
			response += fmt.Sprintf("Pending Updates: %d\n", webhookInfo.PendingUpdateCount)

			if webhookInfo.LastErrorDate > 0 {
				errorTime := time.Unix(int64(webhookInfo.LastErrorDate), 0)
				response += fmt.Sprintf("Last Error: [%s] %s\n",
					errorTime.Format("2006-01-02 15:04:05"),
					webhookInfo.LastErrorMessage)
			}
		} else {
			response += fmt.Sprintf("\nError getting webhook info: %v", err)
		}

		w.Write([]byte(response))
	})

	// 创建一个服务器结构体
	server := &http.Server{
		Addr:    webhookListen,
		Handler: mux,
	}

	// Set up updates handler via webhook
	updates, err := bot.UpdatesViaWebhook(ctx,
		telego.WebhookHTTPServeMux(mux, webhookPath, secretToken),
	)
	if err != nil {
		log.Fatalf("Failed to get updates channel: %v", err)
	}

	// Setup handler
	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatalf("Failed to create bot handler: %v", err)
	}
	defer bh.Stop()

	// 添加一个通用处理程序来记录所有收到的更新
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		log.Printf("Received update: %+v", update)
		return ctx.Next(update) // 继续传递给下一个处理程序
	})

	// 配置消息处理程序，但尚未启动
	// Handle new chat members
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		log.Printf("Processing message: %+v", message)
		if message.From != nil && message.From.IsPremium {
			log.Printf("Found premium user: %s", message.From.FirstName)
			bot.DeleteMessage(ctx.Context(), &telego.DeleteMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: message.MessageID,
			})
			restrictUser(ctx.Context(), bot, message.Chat.ID, message.From.ID)
			sendWarning(ctx.Context(), bot, message.Chat.ID, *message.From)
			return nil
		}

		if message.NewChatMembers != nil {
			log.Printf("New chat members detected: %d members", len(message.NewChatMembers))
			for _, newMember := range message.NewChatMembers {
				// Skip bots
				if newMember.IsBot {
					log.Printf("Skipping bot: %s", newMember.FirstName)
					continue
				}

				// Check if user should be restricted
				if shouldRestrictUser(newMember) {
					log.Printf("Restricting user: %s", newMember.FirstName)
					restrictUser(ctx.Context(), bot, message.Chat.ID, newMember.ID)
					sendWarning(ctx.Context(), bot, message.Chat.ID, newMember)
				}
			}
		}
		return nil
	})

	// 先启动 HTTP 服务器
	go func() {
		log.Printf("Starting HTTP server on %s", webhookListen)
		log.Printf("Bot webhook path: %s, Debug path: /debug", webhookPath)

		// 启动服务器后释放等待信号
		go func() {
			// 给服务器一点时间启动
			time.Sleep(500 * time.Millisecond)
			serverReady.Done()
		}()

		var err error
		if certFile != "" && keyFile != "" {
			log.Printf("Using TLS with cert: %s, key: %s", certFile, keyFile)
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Printf("WARNING: Running without TLS. Make sure you have a HTTPS proxy in front of this server")
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 等待服务器准备就绪
	serverReady.Wait()
	log.Println("HTTP server is ready, starting bot handler...")

	// 在服务器准备就绪后，启动 bot 处理程序
	bh.Start()

	// 替换 select {} 为正确的信号处理逻辑
	// 创建一个通道用于接收操作系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	log.Printf("Received signal: %v, shutting down...", sig)

	// 优雅地关闭服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// shouldRestrictUser checks if a user should be restricted based on their name and username
func shouldRestrictUser(user telego.User) bool {
	// Check for emoji in name
	if hasEmoji(user.FirstName) || hasEmoji(user.LastName) {
		return true
	}

	// Check for random username
	// if isRandomUsername(user.Username) {
	// 	return true
	// }

	return user.IsPremium
}

// hasEmoji checks if a string contains emoji characters
func hasEmoji(s string) bool {
	if s == "" {
		return false
	}
	return emojiRegex.MatchString(s)
}

// isRandomUsername checks if a username appears to be a random string
func isRandomUsername(username string) bool {
	if username == "" {
		return false
	}

	return false
}

// restrictUser restricts a user's permissions in a chat
func restrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
	// Create chat permissions that restrict sending messages and media
	canSendMessages := false
	canSendMedia := false
	canSendPolls := false
	canSendOther := false
	canAddWebPreview := false

	permissions := telego.ChatPermissions{
		CanSendMessages:       &canSendMessages,
		CanSendAudios:         &canSendMedia,
		CanSendDocuments:      &canSendMedia,
		CanSendPhotos:         &canSendMedia,
		CanSendVideos:         &canSendMedia,
		CanSendVideoNotes:     &canSendMedia,
		CanSendVoiceNotes:     &canSendMedia,
		CanSendPolls:          &canSendPolls,
		CanSendOtherMessages:  &canSendOther,
		CanAddWebPagePreviews: &canAddWebPreview,
	}

	// Create restriction config
	params := telego.RestrictChatMemberParams{
		ChatID:      telego.ChatID{ID: chatID},
		UserID:      userID,
		Permissions: permissions,
		UntilDate:   0, // 0 means restrict indefinitely
	}

	// Apply restriction
	err := bot.RestrictChatMember(ctx, &params)
	if err != nil {
		log.Printf("Error restricting user %d: %v", userID, err)
	} else {
		log.Printf("Successfully restricted user %d in chat %d", userID, chatID)
	}
}

// sendWarning sends a warning message about the restricted user to the specified admin
func sendWarning(ctx context.Context, bot *telego.Bot, chatID int64, user telego.User) {
	// Get admin ID from environment variable
	adminIDStr := os.Getenv("TELEGRAM_ADMIN_ID")
	if adminIDStr == "" {
		log.Println("TELEGRAM_ADMIN_ID environment variable not set, not sending notification")
		return
	}

	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid TELEGRAM_ADMIN_ID format: %v", err)
		return
	}

	userName := user.FirstName
	if user.LastName != "" {
		userName += " " + user.LastName
	}

	var reason string
	if hasEmoji(user.FirstName) || hasEmoji(user.LastName) {
		reason = "名称中包含emoji"
	} else if isRandomUsername(user.Username) {
		reason = "用户名是无意义的随机字符串"
	} else if user.IsPremium {
		reason = "用户是Premium用户"
	} else {
		reason = "符合垃圾用户特征"
	}

	// Get group information
	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		log.Printf("Error getting chat info: %v", err)
		return
	}

	// Create message for admin
	groupName := chatInfo.Title
	message := "⚠️ 安全提醒 [" + groupName + "]\n" +
		"用户 " + userName + " 已被限制发送消息和媒体的权限\n" +
		"原因: " + reason

	// Send message to admin
	adminMessageParams := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: adminID},
		Text:   message,
	}

	_, err = bot.SendMessage(ctx, &adminMessageParams)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
	} else {
		log.Printf("Successfully sent restriction notice to admin for user %s", userName)
	}
}
