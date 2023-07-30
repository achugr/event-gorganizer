# Event gorganizer bot

A very minimalistic Telegram bot that helps manage events. The idea is very simple - create an event, then
participants of the chat respond if they join or not. Initially, it was made for organizing football games.

## Commands list

* /i - Add yourself as a participant to the current event. Add name as an argument to add someone.
* /cant - Remove yourself from participants of the current event, pass the position number to remove someone.
* /event - Display the list of participants for the current event.
* /new - Create a new event, only one active event is supported at the moment, creating a new one will close the
  existing one.

# Implementation details

The bot is written in GO to try out the language.

## Infrastructure

The bot uses [GCP Datastore](https://cloud.google.com/datastore).

## Deployment

Deployment is not automated. It's necessary to enable API for datastore and register secrets `TG_KEY` - token provided
by Telegram, and `TG_WEBHOOK_SECRET` - random secret for making webhook url safe. Telegram allows to register webhook
programmatically, so it can be easily rotated.

New service version can be deployed to [GCP Cloudrun](https://cloud.google.com/run) using the cli from the repository
root.

```shell
gcloud run deploy `NAME` --source . --project `PROJECT` --region `REGION` --set-secrets TG_KEY=TG_KEY:latest --set-secrets TG_WEBHOOK_SECR
ET=TG_WEBHOOK_SECRET:latest --allow-unauthenticated

```

Exit code `3` indicates initialization error, check the logs for details. 