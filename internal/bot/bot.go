package tgbot

import (
	"bytes"
	"context"
	_ "embed"
	"event-gorganizer/internal/model"
	"event-gorganizer/internal/service"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type TgBot struct {
	bot                    *tgbotapi.BotAPI
	updates                tgbotapi.UpdatesChannel
	eventService           *service.EventService
	eventRenderingTemplate template.Template
}

func NewPollBot(eventService *service.EventService, tgKey string) *TgBot {
	log.Info().Msg("Starting the bot in poll mode.")
	bot, err := tgbotapi.NewBotAPI(tgKey)
	if err != nil {
		log.Fatal().Msgf("Failed registering the bot: %s.", err)
	}
	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false})
	if err != nil {
		log.Error().Msgf("Deleted a webhook config for the bot.")
	}
	bot.Debug = viper.GetBool("BOT_DEBUG")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	return &TgBot{
		bot:                    bot,
		updates:                bot.GetUpdatesChan(updateConfig),
		eventService:           eventService,
		eventRenderingTemplate: getTemplate(),
	}
}

func NewWebhookBot(eventService *service.EventService, webhookSecret string, tgKey string) *TgBot {
	log.Info().Msg("Starting the bot in webhook mode.")
	bot, err := tgbotapi.NewBotAPI(tgKey)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	bot.Debug = true

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if info.LastErrorDate != 0 {
		log.Error().Msgf("Telegram callback failed: %s.", info.LastErrorMessage)
	}

	updates := bot.ListenForWebhook("/" + webhookSecret)
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal().Msgf("Failed starting the server: %s.", err)
			os.Exit(3)
		}
	}()
	return &TgBot{
		bot:                    bot,
		updates:                updates,
		eventService:           eventService,
		eventRenderingTemplate: getTemplate(),
	}
}

func (b *TgBot) ProcessUpdates() {
	for update := range b.updates {
		ctx := context.Background()

		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "new":
			chatId := update.FromChat().ID
			if b.hasPermissionToCreateEvent(update.SentFrom().ID, update.FromChat().ID) {
				creator := getSelf(update)
				b.eventService.CreateNewEvent(ctx, chatId, creator, update.Message.CommandArguments())
				msg.Text = "Event was created."
			} else {
				msg.Text = "Event wasn't created, not enough rights."
			}
		case "event":
			event := b.eventService.GetActiveEvent(ctx, update.FromChat().ID)
			msg.ParseMode = tgbotapi.ModeHTML
			msg.Text = b.renderEvent(*event)
		case "i":
			self := getSelf(update)
			if hasArguments(update.Message) {
				invitedPerson := update.Message.CommandArguments()
				invitedParticipant := &model.Participant{
					Name:       invitedPerson,
					TelegramId: nil,
					InvitedBy:  self,
				}
				b.eventService.AddNewParticipant(ctx, update.FromChat().ID, invitedParticipant)
				msg.Text = fmt.Sprintf("Person %s was added by %s.", invitedPerson, self.Name)
			} else {
				b.eventService.AddNewParticipant(ctx, update.FromChat().ID, self)
				msg.Text = fmt.Sprintf("Person %s has been added", self.Name)
			}
		case "cant":
			self := getSelf(update)
			if hasArguments(update.Message) {
				participantNumber, err := strconv.Atoi(update.Message.CommandArguments())
				if err != nil {
					msg.Text = "Incorrect participant number."
					return
				}
				removed := b.eventService.RemoveParticipantByNumber(ctx, update.FromChat().ID, participantNumber)
				msg.Text = fmt.Sprintf("Person %s won't attend.", removed.Name)
			} else {
				b.eventService.RemoveParticipant(ctx, update.FromChat().ID, self)
				msg.Text = fmt.Sprintf("Person %s won't attend.", self.Name)
			}
		default:
			msg.Text = fmt.Sprintf("Unknown command: %s.", update.Message.Command())
		}

		if _, err := b.bot.Send(msg); err != nil {
			log.Fatal().Msg(err.Error())
		}
	}
}

func (b *TgBot) hasPermissionToCreateEvent(userId int64, chatId int64) bool {
	resp, err := b.bot.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatId}})
	if err != nil {
		log.Error().Err(err)
	}
	for _, member := range resp {
		if userId == member.User.ID && (member.IsCreator() || member.IsAdministrator()) {
			return true
		}
	}
	return false
}

func getSelf(update tgbotapi.Update) *model.Participant {
	tgUser := update.Message.From
	var name string
	if len(strings.TrimSpace(tgUser.UserName)) > 0 {
		name = tgUser.UserName
	} else if len(strings.TrimSpace(tgUser.FirstName)) > 0 {
		name = fmt.Sprintf("%s %s", tgUser.FirstName, tgUser.LastName)
	} else {
		name = strconv.FormatInt(tgUser.ID, 10)
	}
	return &model.Participant{
		Name:       name,
		TelegramId: &tgUser.ID,
	}
}

func hasArguments(message *tgbotapi.Message) bool {
	return len(strings.TrimSpace(message.CommandArguments())) > 0
}

func (b *TgBot) renderEvent(event model.Event) string {
	var doc bytes.Buffer
	err := b.eventRenderingTemplate.ExecuteTemplate(&doc, "event", event)
	if err != nil {
		log.Error().Msgf("Failed rendering the event %s.", event.Id())
	}
	return doc.String()
}

//go:embed event.gohtml
var templateStr string

func getTemplate() template.Template {
	t, err := template.New("event").Funcs(sprig.FuncMap()).Parse(templateStr)
	if err != nil {
		log.Fatal().Msgf("Failed getting the template: %s.", err)
	}
	return *t
}
