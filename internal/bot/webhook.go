package bot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tg-antispam/internal/logger"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

// WebhookServer represents a webhook HTTP server
type WebhookServer struct {
	server   *http.Server
	certFile string
	keyFile  string
}

// Start starts the webhook server
func (ws *WebhookServer) Start() error {
	logger.Infof("Starting HTTP server on %s", ws.server.Addr)

	// Determine if we should use TLS
	if ws.certFile != "" && ws.keyFile != "" {
		logger.Infof("Using TLS with cert: %s, key: %s", ws.certFile, ws.keyFile)
		return ws.server.ListenAndServeTLS(ws.certFile, ws.keyFile)
	}

	logger.Infof("WARNING: Running without TLS. Make sure you have a HTTPS proxy in front of this server")
	return ws.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (ws *WebhookServer) Shutdown(ctx context.Context) error {
	return ws.server.Shutdown(ctx)
}

// SetupWebhook configures and starts the webhook server
func SetupWebhook(ctx context.Context, bot *telego.Bot, webhookPoint, listenPort, debugPath, secretToken string, certFile, keyFile string) (*th.BotHandler, *WebhookServer, error) {
	if webhookPoint == "" {
		return nil, nil, fmt.Errorf("webhook endpoint is required")
	}

	// Set default values
	if listenPort == "" {
		listenPort = "8443" // Default listen port
		logger.Infof("Using default listen port: %s", listenPort)
	}

	// Validate HTTPS setup
	if (certFile == "" || keyFile == "") && !strings.HasPrefix(webhookPoint, "https://") {
		return nil, nil, fmt.Errorf("HTTPS configuration required: set cert_file and key_file in config or use a HTTPS proxy")
	}

	// Parse URL to get path component
	parsedURL, err := url.Parse(webhookPoint)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid webhook endpoint: %w", err)
	}

	webhookPath := parsedURL.Path
	if webhookPath == "" {
		webhookPath = "/webhook"
		logger.Infof("No path specified in webhook endpoint, using default path: %s", webhookPath)
	}

	// Set up webhook
	logger.Infof("Setting webhook to: %s", webhookPoint)
	setWebhookParams := &telego.SetWebhookParams{
		URL:            webhookPoint,
		AllowedUpdates: []string{"message", "channel_post", "chat_member", "my_chat_member", "callback_query"},
		SecretToken:    secretToken,
	}

	err = bot.SetWebhook(ctx, setWebhookParams)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set webhook: %w", err)
	}

	// Get and display webhook info for debugging
	webhookInfo, err := bot.GetWebhookInfo(ctx)
	if err != nil {
		logger.Infof("Warning: Failed to get webhook info: %v", err)
	} else {
		logger.Infof("Webhook info: URL=%s, HasCustomCert=%v, PendingUpdateCount=%d",
			webhookInfo.URL, webhookInfo.HasCustomCertificate, webhookInfo.PendingUpdateCount)
		if webhookInfo.LastErrorDate > 0 {
			logger.Infof("Webhook last error: [%d] %s", webhookInfo.LastErrorDate, webhookInfo.LastErrorMessage)
		}
		logger.Infof("Allowed updates: %v", webhookInfo.AllowedUpdates)
	}

	// Create HTTP server mux
	mux := http.NewServeMux()

	// Add debug handler
	if debugPath != "" {
		mux.HandleFunc(debugPath, func(w http.ResponseWriter, r *http.Request) {
			logger.Infof("Debug endpoint accessed: %s %s", r.Method, r.URL.Path)

			// Display request headers and content
			logger.Infof("Request headers: %v", r.Header)

			// Return detailed status information
			webhookInfo, err := bot.GetWebhookInfo(ctx)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			botUser, _ := bot.GetMe(ctx)
			response := "Bot webhook server is running\n\n"
			response += fmt.Sprintf("Bot username: %s\n", botUser.Username)
			response += fmt.Sprintf("Webhook path: %s\n", webhookPoint)

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
	}

	webhookListen := "0.0.0.0:" + listenPort
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

	return bh, &WebhookServer{
		server:   server,
		certFile: certFile,
		keyFile:  keyFile,
	}, nil
}
