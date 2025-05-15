package telegram

import (
	"context"
	"strings"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleMessage processes incoming messages
func (b *Bot) handleMessage(message *tgbotapi.Message) {

	if message.IsCommand() {
		b.handleCommand(message)
		return
	}

	// Check if the message text looks like an email
	if utils.IsValidEmail(message.Text) {
		b.handleEmailCheck(message)
		return
	}

	// Respond with help message
	b.sendTranslated(message.Chat.ID, message.From.ID, "invalid_email")
}

// handleCommand handles bot commands
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.sendTranslated(message.Chat.ID, message.From.ID, "welcome")
	case "help":
		b.sendHelpMessage(message.Chat.ID, message.From.ID)
	case "language":
		b.sendLanguageOptions(message.Chat.ID)
	default:
		b.sendTranslated(message.Chat.ID, message.From.ID, "unknown_command")
	}
}

// handleEmailCheck processes email validation and database lookup
func (b *Bot) handleEmailCheck(message *tgbotapi.Message) {
	email := utils.NormalizeEmail(message.Text)

	// Store email in cache for callback handling
	b.emailCache[message.From.ID] = email

	// Check email status
	ctx := context.Background()
	status, user, err := b.service.CheckEmailStatus(ctx, int64(message.From.ID), email)
	if err != nil {
		b.logger.Error("Error checking email status", "email", email, "error", err)
		b.sendTranslated(message.Chat.ID, message.From.ID, "error_occurred")
		return
	}

	switch status {
	case "rate_limited":
		b.sendTranslated(message.Chat.ID, message.From.ID, "rate_limited")
	case "not_found":
		b.sendTranslated(message.Chat.ID, message.From.ID, "email_not_found")
	case "unavailable":
		b.sendTranslated(message.Chat.ID, message.From.ID, "system_unavailable")
	case "redeemed":
		dateStr := user.Redeemed.Format("January 2, 2006")
		b.sendTranslated(message.Chat.ID, message.From.ID, "already_redeemed", "date", dateStr)
	case "eligible":
		b.sendEligibleMessage(message.Chat.ID, message.From.ID)
	default:
		b.sendTranslated(message.Chat.ID, message.From.ID, "error_occurred")
	}
}

// handleCallbackQuery handles button press responses
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Acknowledge the callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	b.api.Request(callback)

	// Handle language selection
	if strings.HasPrefix(query.Data, "lang_") {
		lang := strings.TrimPrefix(query.Data, "lang_")
		b.handleLanguageSelection(query, lang)
		return
	}

	// Get cached email
	email, ok := b.emailCache[query.From.ID]
	if !ok {
		b.sendTranslated(query.Message.Chat.ID, query.From.ID, "email_not_cached")
		return
	}

	switch query.Data {
	case "redeem":
		b.handleRedemption(query, email)
	case "skip":
		b.handleSkip(query)
	default:
		b.sendTranslated(query.Message.Chat.ID, query.From.ID, "error_occurred")
	}

	// Remove buttons from the original message
	b.removeButtons(query.Message)
}

// handleRedemption processes the cocktail redemption
func (b *Bot) handleRedemption(query *tgbotapi.CallbackQuery, email string) {
	ctx := context.Background()
	redemptionTime, err := b.service.RedeemCocktail(ctx, int64(query.From.ID), email)

	if err != nil {
		if err == domain.ErrDatabaseUnavailable {
			b.sendTranslated(query.Message.Chat.ID, query.From.ID, "system_unavailable")
		} else {
			b.logger.Error("Error redeeming cocktail", "email", email, "error", err)
			b.sendTranslated(query.Message.Chat.ID, query.From.ID, "error_occurred")
		}
		return
	}

	dateStr := redemptionTime.Format("January 2, 2006")
	b.sendTranslated(query.Message.Chat.ID, query.From.ID, "redemption_success", "date", dateStr)

	// Remove cached email
	delete(b.emailCache, query.From.ID)
}

// handleSkip processes skipping the cocktail redemption
func (b *Bot) handleSkip(query *tgbotapi.CallbackQuery) {
	b.sendTranslated(query.Message.Chat.ID, query.From.ID, "skip_redemption")

	// Remove cached email
	delete(b.emailCache, query.From.ID)
}

// sendEligibleMessage sends a message with redemption buttons
func (b *Bot) sendEligibleMessage(chatID int64, userID int64) {
	redeemText := b.translate(userID, "button_redeem")
	skipText := b.translate(userID, "button_skip")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(redeemText, "redeem"),
			tgbotapi.NewInlineKeyboardButtonData(skipText, "skip"),
		),
	)

	text := b.translate(userID, "eligible")
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// sendHelpMessage sends help information
func (b *Bot) sendHelpMessage(chatID int64, userID int64) {
	helpText := b.translate(userID, "help_message")
	b.sendMessage(chatID, helpText)
}

// sendLanguageOptions sends a message with language selection buttons
func (b *Bot) sendLanguageOptions(chatID int64) {
	// Get available languages
	languages := b.translator.GetAvailableLanguages()

	// Create language selection keyboard
	var rows [][]tgbotapi.InlineKeyboardButton

	// Map language codes to readable names
	langNames := map[string]string{
		"en": "English",
		"es": "Español",
		"fr": "Français",
		"de": "Deutsch",
		"ru": "Русский",
		"sr": "Српски",
	}

	// Create buttons in groups of 2
	var row []tgbotapi.InlineKeyboardButton
	for i, lang := range languages {
		name, ok := langNames[lang]
		if !ok {
			name = lang // Fallback to code if name not found
		}

		button := tgbotapi.NewInlineKeyboardButtonData(name, "lang_"+lang)
		row = append(row, button)

		// Add row after every 2 buttons or at the end
		if (i+1)%2 == 0 || i == len(languages)-1 {
			rows = append(rows, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, b.translate(0, "language_command"))
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// handleLanguageSelection processes language selection from the user
func (b *Bot) handleLanguageSelection(query *tgbotapi.CallbackQuery, lang string) {
	// Check if language is supported
	supported := false
	for _, l := range b.translator.GetAvailableLanguages() {
		if l == lang {
			supported = true
			break
		}
	}

	if supported {
		// Set user language
		b.setUserLanguage(query.From.ID, lang)

		// First set the language, then translate the confirmation message
		// This ensures the message appears in the newly selected language
		b.sendTranslated(query.Message.Chat.ID, query.From.ID, "language_set")
	} else {
		// Language not supported
		// Show this message in their current language
		b.sendTranslated(query.Message.Chat.ID, query.From.ID, "language_not_supported")
	}

	// Remove buttons from the original message
	b.removeButtons(query.Message)
}

// removeButtons removes the inline keyboard from a message
func (b *Bot) removeButtons(message *tgbotapi.Message) {
	edit := tgbotapi.NewEditMessageReplyMarkup(
		message.Chat.ID,
		message.MessageID,
		tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
		},
	)
	b.api.Send(edit)
}
