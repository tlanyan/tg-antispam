package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

// SetupWebhook configures and starts the webhook server
func SetupWebhook(ctx context.Context, bot *telego.Bot, webhookPoint, webhookPath, webhookListen, secretToken string, certFile, keyFile string) (*th.BotHandler, *http.Server, error) {
	// Set up webhook
	log.Printf("Setting webhook to: %s", webhookPoint)
	setWebhookParams := &telego.SetWebhookParams{
		URL:            webhookPoint,
		AllowedUpdates: []string{"message", "channel_post", "chat_member", "my_chat_member", "callback_query"},
		SecretToken:    secretToken,
	}

	err := bot.SetWebhook(ctx, setWebhookParams)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set webhook: %w", err)
	}

	// Get and display webhook info for debugging
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

	// Add debug handler
	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Debug endpoint accessed: %s %s", r.Method, r.URL.Path)

		// Display request headers and content
		log.Printf("Request headers: %v", r.Header)

		// Return detailed status information
		webhookInfo, err := bot.GetWebhookInfo(ctx)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		botUser, _ := bot.GetMe(ctx)
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

	// Create server struct
	server := &http.Server{
		Addr:    webhookListen,
		Handler: mux,
	}

	// Set up updates handler via webhook
	updates, err := bot.UpdatesViaWebhook(ctx,
		telego.WebhookHTTPServeMux(mux, webhookPath, secretToken),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get updates channel: %w", err)
	}

	// Setup handler
	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create bot handler: %w", err)
	}

	// Configure message handlers using the function from bot_handler.go
	SetupMessageHandlers(bh, bot)

	return bh, server, nil
}
