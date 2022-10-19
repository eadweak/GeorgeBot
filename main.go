package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"log"
	"os"
)

/* загружаем константные значения - Slack Channel ID, Token ID и App Token */

const SlackAuthToken = "xoxb-167583481591-4223371008935-HXq90ylaJRikps07T1Gu0GVR"
const SlackChannelId = "C034N63HHQF"
const SlackAppToken = "xapp-1-A046ZSVBHMY-4238345761154-4c0b661fc284a2defd3b3ad1c699542c797caaf39138a120daab8289d5b4e30d"

// const SlackVerificationToken = "WKpTuCfarLDrMQb8L9HfzOqY"

// определяем Slack-клиент
var client = slack.New(SlackAuthToken, slack.OptionDebug(true), slack.OptionAppLevelToken(SlackAppToken))

func main() {

	/* добавляем текст для сообщения в чате
	attachment := slack.Attachment{}
	attachment.Text = fmt.Sprintf("Я тут, чего изволите?") */

	// выполняем POST-запрос для проверки
	channelId, timestamp, err := client.PostMessage(
		SlackChannelId,
		slack.MsgOptionText("Привет! Я Джордж, ваш дружелюбный спутник по OpsGenie", false),
		// slack.MsgOptionAttachments(attachment), опция может пригодиться
		slack.MsgOptionAsUser(true),
	)

	// проверка на ошибку
	if err != nil {
		log.Fatalf("%s\n", err)
	}

	// логирование ошибки
	log.Printf("Message successfully sent to Channel %s at %s\n", channelId, timestamp)

	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(true),
		// опции для сбора кастомных логов
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	// Create a context that can be used to cancel goroutine
	ctx, cancel := context.WithCancel(context.Background())
	// Make this cancel called properly in a real program , graceful shutdown etc
	defer cancel()

	go func(ctx context.Context, client *slack.Client, socketClient *socketmode.Client) {
		// Create a for loop that selects either the context cancellation or the events incomming
		for {
			select {
			// inscase context cancel is called exit the goroutine
			case <-ctx.Done():
				log.Println("Shutting down socketmode listener")
				return
			case event := <-socketClient.Events:
				// We have a new Events, let's type switch the event
				// Add more use cases here if you want to listen to other events.
				switch event.Type {
				// handle EventAPI events
				case socketmode.EventTypeEventsAPI:
					// The Event sent on the channel is not the same as the EventAPI events so we need to type cast it
					eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
					if !ok {
						log.Printf("Could not type cast the event to the EventsAPIEvent: %v\n", event)
						continue
					}
					// We need to send an Acknowledge to the slack server
					socketClient.Ack(*event.Request)
					// Now we have an Events API event, but this event type can in turn be many types, so we actually need another type switch
					err := handleEventMessage(eventsAPIEvent)
					if err != nil {
						// Replace with actual err handeling
						log.Fatal(err)
					}
				// Handle Slash Commands
				case socketmode.EventTypeSlashCommand:
					// Just like before, type cast to the correct event type, this time a SlashEvent
					command, ok := event.Data.(slack.SlashCommand)
					if !ok {
						log.Printf("Could not type cast the message to a SlashCommand: %v\n", command)
						continue
					}
					// Dont forget to acknowledge the request
					socketClient.Ack(*event.Request)
					// handleSlashCommand will take care of the command
					err := handleSlashCommand(command, client)
					if err != nil {
						log.Fatal(err)
					}

				}
			}

		}
	}(ctx, client, socketClient)

	socketClient.Run()
}

// handleEventMessage завязан на типе ивента
func handleEventMessage(event slackevents.EventsAPIEvent) error {
	switch event.Type {
	// First we check if this is an CallbackEvent
	case slackevents.CallbackEvent:

		innerEvent := event.InnerEvent
		// Yet Another Type switch on the actual Data to see if its an AppMentionEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// The application has been mentioned since this Event is a Mention event
			log.Println(ev)
		}

	default:
		return errors.New("unsupported event type")

	}

	return nil
}

// handleSlashCommand добавляет необходимые нам slash команды
func handleSlashCommand(command slack.SlashCommand, client *slack.Client) error {
	// используем конструкцию switch/case для добавления команд
	switch command.Command {
	case "/getAlerts":
		// команда /getAlerts
		return handleHelloCommand(command, client)
	}

	return nil
}

// handleHelloCommand команда /hello
func handleHelloCommand(command slack.SlashCommand, client *slack.Client) error {
	/* добавляем в поля наш текст и поприветствуем пользователя!
	attachment := slack.Attachment{}
	attachment.Text = fmt.Sprintf("Привет, солнышко!")
	attachment.Color = "#4af030" */

	// отправим сообщение в канал test-opsgeniebot
	// канал должен быть определен для переменной command.ChannelID
	// в PostMessage отправляем сообщение, что мы услышали пользователя

	_, _, err := client.PostMessage(command.ChannelID, slack.MsgOptionText("И тебе привет!", false))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}
