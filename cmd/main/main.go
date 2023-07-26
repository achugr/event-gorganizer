package main

import (
	"cloud.google.com/go/datastore"
	"context"
	tgbot "event-gorganizer/internal/bot"
	"event-gorganizer/internal/repository"
	"event-gorganizer/internal/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
)

func main() {
	initializeLogging()
	viper.AddConfigPath(".")
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Info().Msg("Config file not found, using env variables.")
		} else {
			log.Fatal().Msg("Error on reading the config.")
			os.Exit(3)
		}
	}
	log.Info().Msg("Starting the bot.")
	gcpSettings := getGcpSettings()
	eventRepo := repository.NewEventRepository(context.Background(), gcpSettings)
	eventService := service.NewService(eventRepo)

	bot := createBot(eventService)
	bot.ProcessUpdates()
}

func getGcpSettings() repository.GcpSettings {
	if viper.GetString("ENV") == "LOCAL" {
		gcloudKeyFile := viper.GetString("GCLOUD_KEY_FILE")
		return repository.GcpSettings{
			ProjectName:         viper.GetString("GCP_PROJECT_ID"),
			CredentialsFilePath: &gcloudKeyFile,
		}
	} else {
		return repository.GcpSettings{
			ProjectName: datastore.DetectProjectID,
		}
	}
}

func createBot(eventService *service.EventService) *tgbot.TgBot {
	tgKey := viper.GetString("TG_KEY")
	if viper.GetString("ENV") == "LOCAL" {
		return tgbot.NewPollBot(eventService, tgKey)
	} else {
		webhookSecret := viper.GetString("TG_WEBHOOK_SECRET")
		return tgbot.NewWebhookBot(eventService, webhookSecret, tgKey)
	}
}

func initializeLogging() {
	if viper.GetString("ENV") == "LOCAL" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.LevelFieldName = "severity"
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = zerolog.New(os.Stderr).With().
			Timestamp().
			Logger()
	}
}
