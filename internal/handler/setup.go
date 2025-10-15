package handler

import (
	"context"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/config"
	"tg-antispam/internal/crash"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/service"
)

var globalConfig *config.Config

// 并发控制
var (
	messageProcessingSemaphore chan struct{}
	handlerWaitGroup           sync.WaitGroup
)

func Initialize(cfg *config.Config) {
	globalConfig = cfg
	service.Initialize(cfg)

	// 初始化并发控制，限制同时处理的消息数量
	maxConcurrentMessages := cfg.Bot.MaxConcurrentMessages
	if maxConcurrentMessages <= 0 {
		maxConcurrentMessages = 100 // 默认最大并发数
	}
	messageProcessingSemaphore = make(chan struct{}, maxConcurrentMessages)

	ClearRecentUsers()
}

// SetupMessageHandlers configures all bot message and update handlers
func SetupMessageHandlers(bh *th.BotHandler, bot *telego.Bot) {
	// 启动状态监控
	StartStatusMonitoring()

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		// 异步处理消息
		processMessageAsync(bot, message, "message")
		return nil
	})

	bh.HandleChannelPost(func(ctx *th.Context, message telego.Message) error {
		// 异步处理频道消息
		processMessageAsync(bot, message, "channel_post")
		return nil
	})

	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		// 异步处理群成员更新
		processChatMemberUpdateAsync(bot, update)
		return nil
	}, th.AnyChatMember())

	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		// 异步处理机器人成员状态更新
		processMyChatMemberUpdateAsync(bot, update)
		return nil
	}, th.AnyMyChatMember())

	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		// 异步处理回调查询
		processCallbackQueryAsync(bot, query)
		return nil
	})
}

// processMessageAsync 异步处理消息
func processMessageAsync(bot *telego.Bot, message telego.Message, msgType string) {
	handlerWaitGroup.Add(1)

	crash.SafeGoroutine("message-handler-"+msgType, func() {
		defer handlerWaitGroup.Done()

		// 获取信号量
		select {
		case messageProcessingSemaphore <- struct{}{}:
			defer func() { <-messageProcessingSemaphore }()
		case <-time.After(5 * time.Second):
			logger.Warningf("Message processing timeout, dropping message from chat %d", message.Chat.ID)
			return
		}

		// 设置处理超时
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan bool, 1)
		var err error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("Panic in message processing: %v", r)
					incrementCounter(&totalErrors)
				}
				done <- true
			}()

			// 处理命令
			ok, cmdErr := HandleCommand(bot, message)
			if ok {
				err = cmdErr
				if err != nil {
					incrementCounter(&totalErrors)
				}
				return
			}

			// 处理普通消息
			err = handleIncomingMessage(bot, message)
			if err != nil {
				incrementCounter(&totalErrors)
			} else {
				incrementCounter(&totalMessagesProcessed)
			}
		}()

		select {
		case <-done:
			if err != nil {
				logger.Warningf("Error processing message: %v", err)
			}
		case <-ctx.Done():
			logger.Warningf("Message processing timeout for chat %d", message.Chat.ID)
			incrementCounter(&totalTimeouts)
		}
	})
}

// processChatMemberUpdateAsync 异步处理群成员更新
func processChatMemberUpdateAsync(bot *telego.Bot, update telego.Update) {
	handlerWaitGroup.Add(1)

	crash.SafeGoroutine("chat-member-update-handler", func() {
		defer handlerWaitGroup.Done()

		// 获取信号量
		select {
		case messageProcessingSemaphore <- struct{}{}:
			defer func() { <-messageProcessingSemaphore }()
		case <-time.After(5 * time.Second):
			logger.Warningf("Chat member update processing timeout")
			return
		}

		// 设置处理超时
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan bool, 1)
		var err error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("Panic in chat member update processing: %v", r)
					incrementCounter(&totalErrors)
				}
				done <- true
			}()

			err = handleChatMemberUpdate(bot, update)
			if err != nil {
				incrementCounter(&totalErrors)
			} else {
				incrementCounter(&totalChatMemberUpdates)
			}
		}()

		select {
		case <-done:
			if err != nil {
				logger.Warningf("Error processing chat member update: %v", err)
			}
		case <-ctx.Done():
			logger.Warningf("Chat member update processing timeout")
		}
	})
}

// processMyChatMemberUpdateAsync 异步处理机器人成员状态更新
func processMyChatMemberUpdateAsync(bot *telego.Bot, update telego.Update) {
	handlerWaitGroup.Add(1)

	crash.SafeGoroutine("my-chat-member-update-handler", func() {
		defer handlerWaitGroup.Done()

		// 获取信号量
		select {
		case messageProcessingSemaphore <- struct{}{}:
			defer func() { <-messageProcessingSemaphore }()
		case <-time.After(5 * time.Second):
			logger.Warningf("My chat member update processing timeout")
			return
		}

		// 设置处理超时
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan bool, 1)
		var err error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("Panic in my chat member update processing: %v", r)
				}
				done <- true
			}()

			err = handleMyChatMemberUpdate(bot, update)
		}()

		select {
		case <-done:
			if err != nil {
				logger.Warningf("Error processing my chat member update: %v", err)
			}
		case <-ctx.Done():
			logger.Warningf("My chat member update processing timeout")
		}
	})
}

// processCallbackQueryAsync 异步处理回调查询
func processCallbackQueryAsync(bot *telego.Bot, query telego.CallbackQuery) {
	handlerWaitGroup.Add(1)

	crash.SafeGoroutine("callback-query-handler", func() {
		defer handlerWaitGroup.Done()

		// 获取信号量
		select {
		case messageProcessingSemaphore <- struct{}{}:
			defer func() { <-messageProcessingSemaphore }()
		case <-time.After(5 * time.Second):
			logger.Warningf("Callback query processing timeout")
			return
		}

		// 设置处理超时
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan bool, 1)
		var err error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("Panic in callback query processing: %v", r)
					incrementCounter(&totalErrors)
				}
				done <- true
			}()

			err = HandleCallbackQuery(bot, query)
			if err != nil {
				incrementCounter(&totalErrors)
			} else {
				incrementCounter(&totalCallbackQueries)
			}
		}()

		select {
		case <-done:
			if err != nil {
				logger.Warningf("Error processing callback query: %v", err)
			}
		case <-ctx.Done():
			logger.Warningf("Callback query processing timeout")
		}
	})
}

// WaitForHandlers 等待所有处理器完成（用于优雅关闭）
func WaitForHandlers() {
	handlerWaitGroup.Wait()
}

// GetActiveHandlersCount 获取当前活跃的处理器数量
func GetActiveHandlersCount() int {
	return len(messageProcessingSemaphore)
}
