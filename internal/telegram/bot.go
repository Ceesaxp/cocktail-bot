package telegram

import (
	"fmt"
	"sync"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/i18n"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BotAPI is an interface wrapper for the Telegram Bot API
type BotAPI interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	StopReceivingUpdates()
}

// ServiceInterface defines the methods expected from a service
type ServiceInterface interface {
	CheckEmailStatus(ctx any, userID int64, email string) (string, *domain.User, error)
	RedeemCocktail(ctx any, userID int64, email string) (time.Time, error)
	Close() error
}

// TranslatorInterface defines the methods expected from a translator
type TranslatorInterface interface {
	T(lang, key string, args ...string) string
	DetectLanguage(langCode string) string
	GetAvailableLanguages() []string
	GetFallbackLanguage() string
}

// Bot represents a Telegram bot
type Bot struct {
	api        BotAPI
	service    ServiceInterface
	logger     *logger.Logger
	running    bool
	waitGroup  sync.WaitGroup
	stopCh     chan struct{}
	emailCache map[int64]string     // Map of userID -> last email checked
	translator TranslatorInterface  // Translator for multi-language support
	userLangs  map[int64]string     // Map of userID -> preferred language
}

// New creates a new Telegram bot with the provided API and service
func New(api any, service any, logger *logger.Logger, cfg *config.Config) *Bot {
	// Create translator with config settings
	translator := i18n.NewWithConfig(cfg)
	i18n.LoadDefaultTranslations(translator)

	return &Bot{
		api:        api.(BotAPI),
		service:    service.(ServiceInterface),
		logger:     logger,
		stopCh:     make(chan struct{}),
		emailCache: make(map[int64]string),
		translator: translator,
		userLangs:  make(map[int64]string),
	}
}

// NewFromToken creates a new Telegram bot from a token
func NewFromToken(token string, service *service.Service, logger *logger.Logger, cfg *config.Config) (*Bot, error) {
	// Create bot API
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	// Create translator with config settings
	translator := i18n.NewWithConfig(cfg)
	i18n.LoadDefaultTranslations(translator)

	return &Bot{
		api:        api,
		service:    service,
		logger:     logger,
		stopCh:     make(chan struct{}),
		emailCache: make(map[int64]string),
		translator: translator,
		userLangs:  make(map[int64]string),
	}, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	if b.running {
		return fmt.Errorf("bot is already running")
	}

	b.running = true

	// Get bot info
	botAPI, ok := b.api.(*tgbotapi.BotAPI)
	if ok {
		b.logger.Info("Bot started", "username", botAPI.Self.UserName)
	} else {
		b.logger.Info("Bot started")
	}

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

	// Detect language from user if applicable
	if update.Message != nil && update.Message.From != nil {
		b.detectUserLanguage(update.Message.From)
	} else if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		b.detectUserLanguage(update.CallbackQuery.From)
	}

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

// getUserLanguage gets the user's preferred language
func (b *Bot) getUserLanguage(userID int64) string {
	if lang, ok := b.userLangs[userID]; ok {
		return lang
	}
	// Return the default language from translator's fallback
	return b.translator.GetFallbackLanguage()
}

// detectUserLanguage attempts to detect user's language from Telegram
// and stores it if not already set
func (b *Bot) detectUserLanguage(user *tgbotapi.User) {
	if user == nil || user.LanguageCode == "" {
		return
	}

	// Only detect if language not already set
	if _, exists := b.userLangs[user.ID]; !exists {
		detectedLang := b.detectLanguage(user.LanguageCode)
		b.setUserLanguage(user.ID, detectedLang)
	}
}

// setUserLanguage sets the user's preferred language
func (b *Bot) setUserLanguage(userID int64, lang string) {
	b.userLangs[userID] = lang
}

// detectLanguage attempts to detect the user's language from Telegram
func (b *Bot) detectLanguage(tgLangCode string) string {
	return b.translator.DetectLanguage(tgLangCode)
}

// translate translates a message key for a specific user
func (b *Bot) translate(userID int64, key string, args ...string) string {
	lang := b.getUserLanguage(userID)
	return b.translator.T(lang, key, args...)
}

// sendTranslated sends a translated message to a chat
func (b *Bot) sendTranslated(chatID int64, userID int64, key string, args ...string) {
	text := b.translate(userID, key, args...)
	b.sendMessage(chatID, text)
}

// SetTranslations overrides translations with fixed values for testing
func (b *Bot) SetTranslations(translations map[string]string) {
	// Create a simple mock translator
	b.translator = &mockTranslator{translations: translations}
}

// mockTranslator is a simple mock translator for testing
type mockTranslator struct {
	translations map[string]string
}

func (t *mockTranslator) T(lang, key string, args ...string) string {
	if text, ok := t.translations[key]; ok {
		return text
	}
	return key
}

func (t *mockTranslator) DetectLanguage(langCode string) string {
	return "en"
}

func (t *mockTranslator) GetAvailableLanguages() []string {
	return []string{"en"}
}

func (t *mockTranslator) GetFallbackLanguage() string {
	return "en"
}

// HandleMessage exposes the handleMessage method for testing
func (b *Bot) HandleMessage(message *tgbotapi.Message) {
	b.handleMessage(message)
}

// HandleCommand exposes the handleCommand method for testing
func (b *Bot) HandleCommand(message *tgbotapi.Message) {
	b.handleCommand(message)
}

// HandleCallbackQuery exposes the handleCallbackQuery method for testing
func (b *Bot) HandleCallbackQuery(query *tgbotapi.CallbackQuery) {
	b.handleCallbackQuery(query)
}
