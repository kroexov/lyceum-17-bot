package botsrv

import (
	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"
	"context"
	"encoding/json"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"strings"
)

const (
	startCommand  = "/start"
	patternRole   = "role_"
	patternAction = "action_"
	RoleStudent   = "student"
	RoleGraduate  = "graduate"

	linkRegisterStudent  = "https://forms.gle/fgx4LbfDJBt3qtg3A"
	linkRegisterGraduate = "https://forms.gle/HCc4vYxVEU4YZdp68"
)

var startReplyMarkup = &models.InlineKeyboardMarkup{
	InlineKeyboard: [][]models.InlineKeyboardButton{
		{
			{Text: "Ученик лицея", CallbackData: patternRole + RoleStudent},
			{Text: "Выпускник лицея", CallbackData: patternRole + RoleGraduate},
		},
	},
}

type Config struct {
	Token       string
	AdminChatId int
}

type BotManager struct {
	embedlog.Logger
	dbo db.DB
	cfg Config
}

func NewBotManager(logger embedlog.Logger, dbo db.DB, cfg Config) *BotManager {
	return &BotManager{
		Logger: logger,
		dbo:    dbo,
		cfg:    cfg,
	}
}

func (bm *BotManager) RegisterBotHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, startCommand, bot.MatchTypePrefix, bm.StartHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, patternRole, bot.MatchTypePrefix, bm.RoleChooseHandler)
}

func (bm *BotManager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Chat.Type != models.ChatTypePrivate {
		return
	}
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Привет! Напиши /start чтобы начать!",
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Привет! Выбери кто ты",
		ReplyMarkup: startReplyMarkup,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) RoleChooseHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	var link string

	parts := strings.Split(update.CallbackQuery.Data, "_")
	switch parts[1] {
	case RoleStudent:
		link = linkRegisterStudent
	case RoleGraduate:
		link = linkRegisterGraduate
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Text:      "Пожалуйста, заполни данные о себе в этой форме. Мы принимаем в чат только по заявкам и не анонимно",
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "Пройти регистрацию",
				// todo: + strconv.FormatInt(update.CallbackQuery.From.ID, 10)
				URL: link,
			}}}},
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}
}

func (bm *BotManager) ModerationStudent(ctx context.Context, b *bot.Bot, update *models.Update) {
	var result StudentForm
	if err := json.Unmarshal([]byte(update.CallbackQuery.Data), &result); err != nil {
		bm.Errorf("Ошибка парсинга JSON: %v\nДанные: %s", err, update.CallbackQuery.Data)

		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: bm.cfg.AdminChatId,
			Text:   "Ошибка обработки данных лицеиста",
		}); err != nil {
			bm.Errorf("Ошибка отправки сообщения: %v", err)
		}

		return
	}

	userID := result.TgId

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "Принять",
					CallbackData: patternAction + "accept_" + userID + "_" + RoleStudent,
				},
			},
			{
				{
					Text:         "Отклонить",
					CallbackData: patternAction + "reject_" + userID + "_" + RoleStudent,
				},
			},
		},
	}

	res, err := parseStudent(result)
	if err != nil {
		bm.Errorf("Ошибка обработки данных лицеиста: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      bm.cfg.AdminChatId,
		Text:        res,
		ParseMode:   "Markdown",
		ReplyMarkup: kb,
	})
	if err != nil {
		bm.Errorf("Ошибка отправки сообщения: %v", err)
	}
}

func (bm *BotManager) ModerationGraduate(ctx context.Context, b *bot.Bot, update *models.Update) {
	var result GraduateForm
	if err := json.Unmarshal([]byte(update.CallbackQuery.Data), &result); err != nil {
		bm.Errorf("Ошибка парсинга JSON: %v\nДанные: %s", err, update.CallbackQuery.Data)

		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: bm.cfg.AdminChatId,
			Text:   "Ошибка обработки данных выпускника",
		}); err != nil {
			bm.Errorf("Ошибка отправки сообщения: %v", err)
		}

		return
	}

	userID := result.TgId

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "Принять",
					CallbackData: patternAction + "accept_" + userID + "_" + RoleGraduate,
				},
			},
			{
				{
					Text:         "Отклонить",
					CallbackData: patternAction + "reject_" + userID + "_" + RoleGraduate,
				},
			},
		},
	}

	res, err := parseGraduate(result)
	if err != nil {
		bm.Errorf("Ошибка обработки данных выпускника: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      bm.cfg.AdminChatId,
		Text:        res,
		ParseMode:   "Markdown",
		ReplyMarkup: kb,
	})
	if err != nil {
		bm.Errorf("Ошибка отправки сообщения: %v", err)
	}
}
