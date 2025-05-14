package telegram

import (
	"fmt"
	"sync"

	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot represents a Telegram bot
type Bot struct {
	api        *tgbotapi.BotAPI
	service    *service.Service
	logger     *logger.Logger
	running    bool
	waitGroup  sync.WaitGroup
	stopCh     chan struct{}
	emailCache map[int64]string // Map of userID -> last email checked
}

// New creates a new Telegram bot
func New(token string, service *service.Service, logger *logger.Logger) (*Bot, error) {
	// Create bot API
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &Bot{
		api:        api,
		service:    service,
		logger:     logger,
		stopCh:     make(chan struct{}),
		emailCache: make(map[int64]string),
	}, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	if b.running {
		return fmt.Errorf("bot is already running")
	}

	b.running = true

	// Get bot info
	b.logger.Info("Bot started", "username", b.api.Self.UserName)

	// Get updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	b.waitGroup.Add(1)
	go func() {
		defer b.waitGroup.Done()
		b.processUpdates(updates)
	}()

	return nil
}

// Stop stops the bot
func (b *Bot) Stop() {
	if !b.running {
		return
	}

	b.running = false
	close(b.stopCh)
	b.api.StopReceivingUpdates()
	b.waitGroup.Wait()

	b.logger.Info("Bot stopped")
}

// processUpdates processes updates from Telegram
func (b *Bot) processUpdates(updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-b.stopCh:
			return
		case update, ok := <-updates:
			if !ok {
				return
			}

			// Process the update
			go b.handleUpdate(update)
		}
	}
}

// handleUpdate handles a single update
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Recovered from panic in handleUpdate", "panic", r)
		}
	}()

	if update.Message != nil {
		b.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
	}
}

// sendMessage sends a text message to a chat
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.Error("Error sending message", "chat_id", chatID, "error", err)
	}
}
