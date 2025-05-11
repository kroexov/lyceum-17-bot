package botsrv

import (
	"botsrv/pkg/db"
	"botsrv/pkg/embedlog"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"strings"
)

const (
	startCommand  = "/start"
	patternRole   = "role_"
	patternAction = "action"
	actionAccept  = "accept"
	actionReject  = "reject"
	RoleStudent   = "student"
	RoleGraduate  = "graduate"

	linkRegisterStudent  = "https://docs.google.com/forms/d/e/1FAIpQLSe_k7fTqytGhSY23jorfXC6HnZy79GR7Acr2JGpKn_UJS3hYg/viewform?usp=pp_url&entry.1409108157=%s&entry.433449939=%d"
	linkRegisterGraduate = "https://docs.google.com/forms/d/e/1FAIpQLSelgO9-5K_ug_anDOdzf5gbLmetCfgqm2SsZn26Up8QriLRnA/viewform?usp=pp_url&entry.1052289244=%s&entry.1561674486=%d"
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
	Token        string
	AdminChatId  int
	LyceumChatId int
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
	b.RegisterHandler(bot.HandlerTypeMessageText, startCommand, bot.MatchTypePrefix, bm.PrivateOnly(bm.StartHandler))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, patternRole, bot.MatchTypePrefix, bm.RoleChooseHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, patternAction, bot.MatchTypePrefix, bm.ModerationResultHandler)
}

func (bm BotManager) PrivateOnly(handler bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {

		if update.Message != nil && update.Message.Chat.Type != "private" {
			return
		}

		handler(ctx, b, update)
	}
}

func (bm *BotManager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
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

	member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
		ChatID: bm.cfg.LyceumChatId,
		UserID: update.Message.From.ID,
	})
	if err != nil {
		bm.Errorf("%v", err)
		return
	}

	if member != nil && member.Left == nil {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Привет! Ты уже состоишь в группе лицея!",
		})
		if err != nil {
			bm.Errorf("%v", err)
			return
		}
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
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
		link = fmt.Sprintf(linkRegisterStudent, update.CallbackQuery.From.Username, update.CallbackQuery.From.ID)
	case RoleGraduate:
		link = fmt.Sprintf(linkRegisterGraduate, update.CallbackQuery.From.Username, update.CallbackQuery.From.ID)
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
					CallbackData: strings.Join([]string{patternAction, actionAccept, userID, RoleStudent}, "_"),
				},
			},
			{
				{
					Text:         "Отклонить",
					CallbackData: strings.Join([]string{patternAction, actionReject, userID, RoleStudent}, "_"),
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
					CallbackData: strings.Join([]string{patternAction, actionAccept, userID, RoleGraduate}, "_"),
				},
			},
			{
				{
					Text:         "Отклонить",
					CallbackData: strings.Join([]string{patternAction, actionReject, userID, RoleGraduate}, "_"),
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
		ReplyMarkup: kb,
	})
	if err != nil {
		bm.Errorf("Ошибка отправки сообщения: %v", err)
	}
}

func (bm *BotManager) ModerationResultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	parts := strings.Split(update.CallbackQuery.Data, "_")
	if len(parts) < 4 {
		bm.Errorf("len parts < 4")
		return
	}

	// todo role stuff
	action, userId, _ := parts[1], parts[2], parts[3]

	switch action {
	case actionAccept:
		link, err := b.CreateChatInviteLink(ctx, &bot.CreateChatInviteLinkParams{
			ChatID:      bm.cfg.LyceumChatId,
			Name:        "Ссылка на вступление в чат с выпускниками",
			MemberLimit: 1,
		})
		if err != nil {
			bm.Errorf("Ошибка отправки сообщения: %v", err)
			return
		}

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userId,
			Text:   "Ваша заявка была принята! Вот одноразовая ссылка на вступление в группу:\n" + link.InviteLink,
		})
		if err != nil {
			bm.Errorf("Ошибка отправки сообщения: %v", err)
		}

		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			Text:        "Заявка принята!\n\n" + update.CallbackQuery.Message.Message.Text,
			ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
			MessageID:   update.CallbackQuery.Message.Message.ID,
			ReplyMarkup: nil,
		})
		if err != nil {
			bm.Errorf("Ошибка исправления сообщения: %v", err)
			return
		}

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			Text:            strings.ReplaceAll(update.CallbackQuery.Message.Message.Text, "Новая заявка от выпускника!", "Новый выпускник!"),
			ChatID:          bm.cfg.LyceumChatId,
			MessageThreadID: 8,
			ReplyMarkup:     nil,
		})
		if err != nil {
			bm.Errorf("Ошибка исправления сообщения: %v", err)
			return
		}

	case actionReject:
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userId,
			Text:   "Ваша заявка была отклонена! Свяжитесь с @kroexov или @mikhailpuminov, если есть вопросы.",
		})
		if err != nil {
			bm.Errorf("Ошибка отправки сообщения: %v", err)
		}

		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			Text:        "Заявка отклонена!\n\n" + update.CallbackQuery.Message.Message.Text,
			ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
			MessageID:   update.CallbackQuery.Message.Message.ID,
			ReplyMarkup: nil,
		})
		if err != nil {
			bm.Errorf("Ошибка исправления сообщения: %v", err)
			return
		}
	}
}
