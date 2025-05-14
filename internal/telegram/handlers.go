package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ceesaxp/cocktail-bot/internal/domain"
	"github.com/Ceesaxp/cocktail-bot/internal/utils"
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
	b.sendMessage(message.Chat.ID, "That doesn't look like a valid email address. Please send a properly formatted email (e.g., example@domain.com).")
}

// handleCommand handles bot commands
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.sendMessage(message.Chat.ID, "Welcome to the Cocktail Bot! Send your email to check if you're eligible for a free cocktail.")
	case "help":
		b.sendHelpMessage(message.Chat.ID)
	default:
		b.sendMessage(message.Chat.ID, "Unknown command. Please send your email to check eligibility or use /help for more information.")
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
		b.sendMessage(message.Chat.ID, "Sorry, an error occurred. Please try again later.")
		return
	}

	switch status {
	case "rate_limited":
		b.sendMessage(message.Chat.ID, "You've made too many requests. Please try again in a few minutes.")
	case "not_found":
		b.sendMessage(message.Chat.ID, "Email is not in database.")
	case "unavailable":
		b.sendMessage(message.Chat.ID, "Sorry, our system is temporarily unavailable. Please try again later.")
	case "redeemed":
		b.sendMessage(message.Chat.ID, fmt.Sprintf("Email found, but free cocktail already consumed on %s.", 
			user.AlreadyConsumed.Format("January 2, 2006")))
	case "eligible":
		b.sendEligibleMessage(message.Chat.ID)
	default:
		b.sendMessage(message.Chat.ID, "Sorry, an error occurred. Please try again later.")
	}
}

// handleCallbackQuery handles button press responses
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Acknowledge the callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	b.api.Request(callback)

	// Get cached email
	email, ok := b.emailCache[query.From.ID]
	if !ok {
		b.sendMessage(query.Message.Chat.ID, "Sorry, I can't find your email. Please try again.")
		return
	}

	switch query.Data {
	case "redeem":
		b.handleRedemption(query, email)
	case "skip":
		b.handleSkip(query)
	default:
		b.sendMessage(query.Message.Chat.ID, "Unknown action. Please try again.")
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
			b.sendMessage(query.Message.Chat.ID, "Sorry, our system is temporarily unavailable. Please try again later.")
		} else {
			b.logger.Error("Error redeeming cocktail", "email", email, "error", err)
			b.sendMessage(query.Message.Chat.ID, "Sorry, an error occurred. Please try again later.")
		}
		return
	}

	b.sendMessage(query.Message.Chat.ID, fmt.Sprintf("Enjoy your free cocktail! Redeemed on %s.", 
		redemptionTime.Format("January 2, 2006")))

	// Remove cached email
	delete(b.emailCache, query.From.ID)
}

// handleSkip processes skipping the cocktail redemption
func (b *Bot) handleSkip(query *tgbotapi.CallbackQuery) {
	b.sendMessage(query.Message.Chat.ID, "You've chosen to skip the cocktail redemption. You can check again later.")

	// Remove cached email
	delete(b.emailCache, query.From.ID)
}

// sendEligibleMessage sends a message with redemption buttons
func (b *Bot) sendEligibleMessage(chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Get Cocktail", "redeem"),
			tgbotapi.NewInlineKeyboardButtonData("Skip", "skip"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "Email found! You're eligible for a free cocktail.")
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// sendHelpMessage sends help information
func (b *Bot) sendHelpMessage(chatID int64) {
	help := `Here's how to use the Cocktail Bot:

• Send your email address to check if you're eligible for a free cocktail
• If eligible, you'll receive options to redeem or skip
• Choose "Get Cocktail" to redeem your free drink
• Each email can only be redeemed once

Commands:
/start - Start the bot
/help - Show this help message

Send an email address to begin!`

	b.sendMessage(chatID, help)
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
