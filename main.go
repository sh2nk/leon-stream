package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/longpoll-bot"
	"github.com/google/uuid"
	"github.com/nicklaw5/helix"

	_ "github.com/lib/pq"
)

var (
	Token          string
	PostgresURL    string
	DB             *sql.DB
	VK             *api.VK
	TableName      string
	TwitchClientID string
	TwitchSecret   string
	BroadcasterID  string
	Domain         string
	SubSecret      string
	Port           string
	EventStrings   Strings
)

func init() {
	Token = getEnv("VK_TOKEN", "t0k3nex4mpl3")
	PostgresURL = getEnv("POSTGRES_URL", "postgres://user:password@localhost:5432/s3cr3t")
	TwitchClientID = getEnv("TWITCH_CLIENT_ID", "fak3idixn8jqlgtr6n045c6plymhir")
	TwitchSecret = getEnv("TWITCH_SECRET", "fak3s3cr37peo88hl2erzjggg0k30c")
	BroadcasterID = getEnv("BROADCASTER_ID", "1337")
	Domain = getEnv("DOMAIN", "https://example.com")
	Port = getEnv("APP_PORT", ":8081")
}

func main() {
	var err error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	EventStrings, err = getStrings("strings.yml")
	if err != nil {
		log.Fatalf("Fatal error during strings.yml parsing")
	}

	log.Println(EventStrings)

	// Creating a new twitch client
	client, err := helix.NewClient(&helix.Options{
		ClientID:     TwitchClientID,
		ClientSecret: TwitchSecret,
	})
	if err != nil {
		log.Fatalf("Falied to create a new Twitch API client, %v\n", err)
	}

	// Requesting a new access token
	apptoken, err := client.RequestAppAccessToken([]string{"user:read:email"})
	if err != nil {
		log.Fatalf("Code %d: Falied to obtain a Twitch API access token, %v\n", apptoken.StatusCode, err)
	}

	// Set the access token on the client
	client.SetAppAccessToken(apptoken.Data.AccessToken)

	// Init new VK api
	VK = api.NewVK(Token)

	ctx := context.Background()

	// Get information about the group
	group, err := VK.GroupsGetByID(nil)
	if err != nil {
		log.Fatalf("Could not obtain groups info: %v\n", err)
	}

	// Init db conection
	DB, err = sql.Open("postgres", PostgresURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer DB.Close()

	// Generate a name for db table
	TableName = fmt.Sprintf("users%d", group[0].ID)
	log.Printf("Current table is: %s\n", TableName)

	// Check db existense
	exists, empty, err := checkDB(ctx)
	if err != nil {
		log.Fatalf("Falied to check db existense: %v\n", err)
	}

	log.Printf("Table exists: %v, Table is empty: %v\n", exists, empty)

	// Create and sync db table if not exists
	if !exists {
		createDB(ctx)
	}
	if empty {
		if err := syncDB(ctx, group[0].ID); err != nil {
			log.Fatalf("Falied to sync db: %v\n", err)
		}
	}

	// Initializing Long Poll
	lp, err := longpoll.NewLongPoll(VK, group[0].ID)
	if err != nil {
		log.Fatal(err)
	}

	// Registering event handlers
	lp.MessageAllow(onMessageAllow) // Allow incoming messages event
	lp.MessageDeny(onMessageDeny)   // Deny incoming messages event
	lp.MessageNew(onMessageNew)     // New message event

	// Runing Bots Long Poll
	log.Println("Starting longpoll...\nBot in now online!")
	go lp.Run()

	SubSecret = uuid.NewString()

	// Creating a new subscription
	respCreate, err := client.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:    helix.EventSubTypeStreamOnline,
		Version: "1",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: BroadcasterID,
		},
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: fmt.Sprint(Domain, "/event"),
			Secret:   SubSecret,
		},
	})
	if err != nil {
		log.Fatalf("Code %d: Error during subscribing to event, %v\n", respCreate.StatusCode, err)
	}
	log.Printf("Code %d: Added a new subscription for broadcaster id%s\n", respCreate.StatusCode, BroadcasterID)

	// Starting event handler
	http.HandleFunc("/event", eventHandler)
	go http.ListenAndServe(Port, nil)
	log.Printf("Event handler listening on port %s", Port)

	// Make the correct shutdown on SIGTERM etc.
	<-shutdown

	_, cancel := context.WithCancel(context.Background())
	defer func(cancel context.CancelFunc) {
		// Removing subscriptions on app exit
		respSubs, err := client.GetEventSubSubscriptions(&helix.EventSubSubscriptionsParams{
			Status: helix.EventSubStatusEnabled, // This is optional.
		})
		if err != nil {
			log.Fatalf("Code %d: Error obtaining subscriptions list, %v\n", respSubs.StatusCode, err)
		}

		respRem, err := client.RemoveEventSubSubscription(respSubs.Data.EventSubSubscriptions[0].ID)
		if err != nil {
			log.Fatalf("Code %d: Error deleting subscriptions, %v\n", respRem.StatusCode, err)
		}
		log.Println("Removed subscription")
		cancel()
	}(cancel)
}
