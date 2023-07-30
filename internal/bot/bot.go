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
	templating "html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type TgBot struct {
	bot                    *tgbotapi.BotAPI
	updates                tgbotapi.UpdatesChannel
	eventService           *service.EventService
	eventRenderingTemplate *templating.Template
}

func NewPollBot(eventService *service.EventService, tgKey string) (*TgBot, error) {
	log.Info().Msg("Starting the bot in poll mode.")
	bot, err := tgbotapi.NewBotAPI(tgKey)
	if err != nil {
		log.Fatal().Msgf("Failed registering the bot: %s.", err)
	}
	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false})
	if err != nil {
		log.Error().Msgf("Failed to delete a webhook config for the bot.")
	}
	bot.Debug = viper.GetBool("BOT_DEBUG")
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	template, err := getTemplate()
	if err != nil {
		return nil, err
	}
	return &TgBot{
		bot:                    bot,
		updates:                bot.GetUpdatesChan(updateConfig),
		eventService:           eventService,
		eventRenderingTemplate: template,
	}, nil
}

func NewWebhookBot(eventService *service.EventService, webhookSecret string, tgKey string) (*TgBot, error) {
	log.Info().Msg("Starting the bot in webhook mode.")
	bot, err := tgbotapi.NewBotAPI(tgKey)
	if err != nil {
		log.Error().Msgf("Failed to initialize the bot: %s", err.Error())
		return nil, err
	}

	bot.Debug = true

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Error().Msg(err.Error())
		return nil, err
	}

	if info.LastErrorDate != 0 {
		log.Error().Msgf("Telegram callback failed: %s.", info.LastErrorMessage)
		return nil, err
	}

	updates := bot.ListenForWebhook("/" + webhookSecret)
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Error().Msgf("Failed to start the server: %s.", err)
			os.Exit(3)
		}
	}()
	template, err := getTemplate()
	if err != nil {
		return nil, err
	}
	return &TgBot{
		bot:                    bot,
		updates:                updates,
		eventService:           eventService,
		eventRenderingTemplate: template,
	}, nil
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

		arguments := update.Message.CommandArguments()
		switch update.Message.Command() {
		case "new":
			chatId := update.FromChat().ID
			hasPermission, err := b.hasPermissionToCreateEvent(update.SentFrom().ID, update.FromChat().ID)
			if err != nil {
				log.Error().Msgf("Failed to check permissions for the chat %s: %s.", chatId, err)
				msg.Text = "Failed to check permissions."
			} else if hasPermission {
				creator := getSelf(update)
				_, err := b.eventService.CreateNewEvent(ctx, chatId, creator, arguments)
				if err != nil {
					log.Error().Msgf("Failed to create an event for the chat %s: %s.", chatId, err)
					msg.Text = "Failed to create an event."
				} else {
					msg.Text = "Event created."
				}
			} else {
				msg.Text = "Event wasn't created, not enough rights."
			}
		case "event":
			event, err := b.eventService.GetActiveEvent(ctx, update.FromChat().ID)
			if err != nil {
				log.Error().Msgf("Failed to get an active event for the chat %s: %s.", update.FromChat().ID, err)
				msg.Text = "Failed to get an active event."
			} else {
				msg.ParseMode = tgbotapi.ModeHTML
				msg.Text = b.renderEvent(*event)
			}
		case "i":
			self := getSelf(update)
			if hasArguments(update.Message) {
				invitedPerson := arguments
				invitedParticipant := &model.Participant{
					Name:       invitedPerson,
					TelegramId: nil,
					InvitedBy:  self,
				}
				invitedParticipant, err := b.eventService.AddNewParticipant(ctx, update.FromChat().ID, invitedParticipant)
				if err != nil {
					log.Error().Msgf("Failed to add %s: %s.", invitedPerson, err)
					msg.Text = fmt.Sprintf("Failed to add %s.", invitedPerson)
				} else {
					msg.Text = fmt.Sprintf("%s added by %s.", invitedPerson, self.Name)
				}
			} else {
				_, err := b.eventService.AddNewParticipant(ctx, update.FromChat().ID, self)
				if err != nil {
					log.Error().Msgf("Failed to add %s: %s.", self.Name, err)
					msg.Text = fmt.Sprintf("Failed to add %s.", self.Name)
				} else {
					msg.Text = fmt.Sprintf("%s added.", self.Name)
				}
			}
		case "cant":
			self := getSelf(update)
			if hasArguments(update.Message) {
				participantNumber, err := strconv.Atoi(arguments)
				if err != nil {
					msg.Text = fmt.Sprintf("Incorrect participant number: %s.", arguments)
					return
				}
				removed, err := b.eventService.RemoveParticipantByNumber(ctx, update.FromChat().ID, participantNumber)
				if err != nil {
					log.Error().Msgf("Failed to remove %d: %s.", participantNumber, err)
					msg.Text = fmt.Sprintf("Failed to remove %d.", participantNumber)
				} else {
					msg.Text = fmt.Sprintf("%s won't attend.", removed.Name)
				}
			} else {
				_, err := b.eventService.RemoveParticipant(ctx, update.FromChat().ID, self)
				if err != nil {
					log.Error().Msgf("Failed to remove %s: %s.", self.Name, err)
					msg.Text = fmt.Sprintf("Failed to remove %s.", self.Name)
				} else {
					msg.Text = fmt.Sprintf("%s won't attend.", self.Name)
				}
			}
		default:
			msg.Text = fmt.Sprintf("Unknown command: %s.", update.Message.Command())
		}

		if _, err := b.bot.Send(msg); err != nil {
			log.Fatal().Msg(err.Error())
		}
	}
}

func (b *TgBot) hasPermissionToCreateEvent(userId int64, chatId int64) (bool, error) {
	resp, err := b.bot.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatId}})
	if err != nil {
		log.Error().Msgf("Failed to get chat administrators: %s.", err)
		return false, err
	}
	for _, member := range resp {
		if userId == member.User.ID && (member.IsCreator() || member.IsAdministrator()) {
			return true, nil
		}
	}
	return false, nil
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
		log.Error().Msgf("Failed to render the event %s.", event.Id())
	}
	return doc.String()
}

//go:embed event.gohtml
var templateStr string

func getTemplate() (*templating.Template, error) {
	t, err := templating.New("event").Funcs(sprig.FuncMap()).Parse(templateStr)
	if err != nil {
		log.Error().Msgf("Failed to get the template: %s.", err)
		return nil, err
	}
	return t, nil
}
