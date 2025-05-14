package telegram_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ceesaxp/cocktail-bot/internal/domain"
	"github.com/Ceesaxp/cocktail-bot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// mockService is a mock implementation of the service
type mockService struct {
	status      string
	user        *domain.User
	redeemError error
}

func (s *mockService) CheckEmailStatus(ctx context.Context, userID int64, email string) (string, *domain.User, error) {
	return s.status, s.user, nil
}

func (s *mockService) RedeemCocktail(ctx context.Context, userID int64, email string) (time.Time, error) {
	if s.redeemError != nil {
		return time.Time{}, s.redeemError
	}
	return time.Now(), nil
}

func (s *mockService) Close() error {
	return nil
}

// mockBotAPI is a mock of the Telegram Bot API
type mockBotAPI struct {
	messagesSent     []tgbotapi.MessageConfig
	callbackAnswers  []tgbotapi.CallbackConfig
	messagesEdited   []tgbotapi.EditMessageReplyMarkupConfig
	updateConfig     tgbotapi.UpdateConfig
	selfUser         tgbotapi.User
	updatesChannel   chan tgbotapi.Update
	receivingUpdates bool
}

func newMockBotAPI() *mockBotAPI {
	return &mockBotAPI{
		messagesSent:     make([]tgbotapi.MessageConfig, 0),
		callbackAnswers:  make([]tgbotapi.CallbackConfig, 0),
		messagesEdited:   make([]tgbotapi.EditMessageReplyMarkupConfig, 0),
		selfUser:         tgbotapi.User{ID: 123, UserName: "testbot", IsBot: true},
		updatesChannel:   make(chan tgbotapi.Update, 100),
		receivingUpdates: false,
	}
}

func (m *mockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	switch v := c.(type) {
	case tgbotapi.MessageConfig:
		m.messagesSent = append(m.messagesSent, v)
		return tgbotapi.Message{MessageID: len(m.messagesSent)}, nil
	case tgbotapi.CallbackConfig:
		m.callbackAnswers = append(m.callbackAnswers, v)
		return tgbotapi.Message{}, nil
	case tgbotapi.EditMessageReplyMarkupConfig:
		m.messagesEdited = append(m.messagesEdited, v)
		return tgbotapi.Message{}, nil
	default:
		return tgbotapi.Message{}, nil
	}
}

func (m *mockBotAPI) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	// For callback responses
	if callback, ok := c.(tgbotapi.CallbackConfig); ok {
		m.callbackAnswers = append(m.callbackAnswers, callback)
	}
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (m *mockBotAPI) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	m.updateConfig = config
	m.receivingUpdates = true
	return m.updatesChannel
}

func (m *mockBotAPI) StopReceivingUpdates() {
	m.receivingUpdates = false
	close(m.updatesChannel)
}

func TestBot(t *testing.T) {
	// Create a mock service
	now := time.Now()
	eligibleUser := &domain.User{
		ID:              "1",
		Email:           "eligible@example.com",
		DateAdded:       now,
		AlreadyConsumed: nil,
	}

	redeemTime := now.Add(-24 * time.Hour)
	redeemedUser := &domain.User{
		ID:              "2",
		Email:           "redeemed@example.com",
		DateAdded:       now,
		AlreadyConsumed: &redeemTime,
	}

	mockSvc := &mockService{
		status:      "eligible",
		user:        eligibleUser,
		redeemError: nil,
	}

	// Create a mock bot API
	mockAPI := newMockBotAPI()

	// Create logger
	l := logger.New("info")

	// Create the bot
	bot := telegram.New(mockAPI, mockSvc, l)

	// Test start command
	bot.HandleCommand(&tgbotapi.Message{
		MessageID: 1,
		From:      &tgbotapi.User{ID: 456, UserName: "testuser"},
		Chat:      &tgbotapi.Chat{ID: 789, Type: "private"},
		Text:      "/start",
		Entities:  []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 6}},
	})

	// Verify start command response
	if len(mockAPI.messagesSent) != 1 {
		t.Fatalf("Expected 1 message sent, got %d", len(mockAPI.messagesSent))
	}

	if mockAPI.messagesSent[0].Text != "Welcome to the Cocktail Bot! Send your email to check if you're eligible for a free cocktail." {
		t.Errorf("Unexpected start command response: %s", mockAPI.messagesSent[0].Text)
	}

	// Test eligible email
	mockAPI.messagesSent = nil // Clear previous messages
	bot.HandleMessage(&tgbotapi.Message{
		MessageID: 2,
		From:      &tgbotapi.User{ID: 456, UserName: "testuser"},
		Chat:      &tgbotapi.Chat{ID: 789, Type: "private"},
		Text:      "eligible@example.com",
	})

	// Verify eligible response (should have inline keyboard)
	if len(mockAPI.messagesSent) != 1 {
		t.Fatalf("Expected 1 message sent, got %d", len(mockAPI.messagesSent))
	}

	if mockAPI.messagesSent[0].Text != "Email found! You're eligible for a free cocktail." {
		t.Errorf("Unexpected eligible response: %s", mockAPI.messagesSent[0].Text)
	}

	if mockAPI.messagesSent[0].ReplyMarkup == nil {
		t.Errorf("Expected reply markup with buttons")
	}

	// Test already redeemed email
	mockSvc.status = "redeemed"
	mockSvc.user = redeemedUser
	mockAPI.messagesSent = nil // Clear previous messages

	bot.HandleMessage(&tgbotapi.Message{
		MessageID: 3,
		From:      &tgbotapi.User{ID: 456, UserName: "testuser"},
		Chat:      &tgbotapi.Chat{ID: 789, Type: "private"},
		Text:      "redeemed@example.com",
	})

	// Verify redeemed response (no buttons)
	if len(mockAPI.messagesSent) != 1 {
		t.Fatalf("Expected 1 message sent, got %d", len(mockAPI.messagesSent))
	}

	if !strings.Contains(mockAPI.messagesSent[0].Text, "Email found, but free cocktail already consumed") {
		t.Errorf("Unexpected redeemed response: %s", mockAPI.messagesSent[0].Text)
	}

	// Test not found email
	mockSvc.status = "not_found"
	mockSvc.user = nil
	mockAPI.messagesSent = nil // Clear previous messages

	bot.HandleMessage(&tgbotapi.Message{
		MessageID: 4,
		From:      &tgbotapi.User{ID: 456, UserName: "testuser"},
		Chat:      &tgbotapi.Chat{ID: 789, Type: "private"},
		Text:      "notfound@example.com",
	})

	// Verify not found response
	if len(mockAPI.messagesSent) != 1 {
		t.Fatalf("Expected 1 message sent, got %d", len(mockAPI.messagesSent))
	}

	if mockAPI.messagesSent[0].Text != "Email is not in database." {
		t.Errorf("Unexpected not found response: %s", mockAPI.messagesSent[0].Text)
	}

	// Test invalid email
	mockAPI.messagesSent = nil // Clear previous messages

	bot.HandleMessage(&tgbotapi.Message{
		MessageID: 5,
		From:      &tgbotapi.User{ID: 456, UserName: "testuser"},
		Chat:      &tgbotapi.Chat{ID: 789, Type: "private"},
		Text:      "not-an-email",
	})

	// Verify invalid email response
	if len(mockAPI.messagesSent) != 1 {
		t.Fatalf("Expected 1 message sent, got %d", len(mockAPI.messagesSent))
	}

	if !strings.Contains(mockAPI.messagesSent[0].Text, "doesn't look like a valid email") {
		t.Errorf("Unexpected invalid email response: %s", mockAPI.messagesSent[0].Text)
	}

	// Test callback query (redeem)
	mockSvc.status = "eligible"  // Reset status
	mockSvc.user = eligibleUser // Reset user
	mockAPI.messagesSent = nil  // Clear previous messages

	// First register email (to populate cache)
	bot.HandleMessage(&tgbotapi.Message{
		MessageID: 6,
		From:      &tgbotapi.User{ID: 456, UserName: "testuser"},
		Chat:      &tgbotapi.Chat{ID: 789, Type: "private"},
		Text:      "eligible@example.com",
	})

	// Then simulate callback
	bot.HandleCallbackQuery(&tgbotapi.CallbackQuery{
		ID:      "callback1",
		From:    &tgbotapi.User{ID: 456, UserName: "testuser"},
		Message: &tgbotapi.Message{MessageID: 6, Chat: &tgbotapi.Chat{ID: 789, Type: "private"}},
		Data:    "redeem",
	})

	// Verify callback responses
	if len(mockAPI.callbackAnswers) == 0 {
		t.Errorf("Expected callback answer")
	}

	if len(mockAPI.messagesSent) != 2 { // One for eligible message, one for redemption
		t.Errorf("Expected 2 messages sent, got %d", len(mockAPI.messagesSent))
	}

	if !strings.Contains(mockAPI.messagesSent[1].Text, "Enjoy your free cocktail") {
		t.Errorf("Unexpected redemption response: %s", mockAPI.messagesSent[1].Text)
	}

	if len(mockAPI.messagesEdited) != 1 {
		t.Errorf("Expected message to be edited to remove buttons")
	}
}
