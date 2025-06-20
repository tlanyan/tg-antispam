package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"tg-antispam/internal/service"
	"time"

	"github.com/mymmrac/telego"
)

// isUserAdmin checks if a user is an admin in a chat
func isUserAdmin(bot *telego.Bot, chatID int64, userID int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admins, err := bot.GetChatAdministrators(ctx, &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		return false
	}

	for _, admin := range admins {
		if admin.MemberUser().ID == userID {
			return true
		}
	}

	return false
}

// getBotUsername retrieves the bot's username
func getBotUsername(bot *telego.Bot) (string, error) {
	botUser, err := bot.GetMe(context.Background())
	if err != nil {
		return "", err
	}
	return botUser.Username, nil
}

func getGroupAndUserID(data string) (int64, int64, error) {
	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		return 0, 0, fmt.Errorf("invalid data format: %s", data)
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid group ID: %v", err)
	}

	userID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid user ID: %v", err)
	}

	return groupID, userID, nil
}

func getLinkedUserName(bot *telego.Bot, userID int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user information
	userInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: userID},
	})
	if err != nil {
		return "", fmt.Errorf("Error getting user info: %v", err)
	}

	userName := userInfo.FirstName
	if userInfo.LastName != "" {
		userName += " " + userInfo.LastName
	}

	// Create user link
	userLink := fmt.Sprintf("tg://user?id=%d", userID)
	return fmt.Sprintf("<a href=\"%s\">%s</a>", userLink, userName), nil
}

// ClassifyWithGemini calls the Gemini Pro API to classify a message as spam.
// Returns true if spam, false otherwise.
func ClassifyWithGemini(apiKey string, model string, message string) (bool, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", model, apiKey)
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": fmt.Sprintf("这是一条群里的聊天消息，请判断是否黄、赌、毒、广告、骚扰或者诈骗信息？仅回复 是 或 否:\n%s", message)},
				},
			},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("gemini API returned status code %d: %s", resp.StatusCode, string(respBytes))
	}
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if len(result.Candidates) == 0 {
		if result.PromptFeedback.BlockReason != "" {
			return true, nil
		}
		return false, fmt.Errorf("no candidates returned from gemini API")
	}
	if len(result.Candidates[0].Content.Parts) == 0 {
		return false, fmt.Errorf("no text parts returned from gemini API")
	}
	text := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	if strings.HasPrefix(text, "是") || strings.HasPrefix(strings.ToLower(text), "yes") {
		return true, nil
	}
	return false, nil
}

func GetBotLang(bot *telego.Bot, message telego.Message) string {
	if message.Chat.Type == "private" {
		return service.GetGroupInfo(bot, message.From.ID, true).Language
	}
	return service.GetGroupInfo(bot, message.Chat.ID, false).Language
}

func GetBotQueryLang(bot *telego.Bot, query *telego.CallbackQuery) string {
	// 首先检查 query 和 query.Message 是否为 nil
	if query == nil || query.Message == nil {
		return "zh_CN"
	}

	if msg, ok := query.Message.(*telego.Message); ok {
		return GetBotLang(bot, *msg)
	}

	return "zh_CN"
}
