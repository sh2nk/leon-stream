package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/SevereCloud/vksdk/v2/events"
	"github.com/nicklaw5/helix"
)

type EventSubNotification struct {
	Subscription helix.EventSubSubscription `json:"subscription"`
	Challenge    string                     `json:"challenge"`
	Event        json.RawMessage            `json:"event"`
}

func onMessageNew(ctx context.Context, obj events.MessageNewObject) {
	log.Printf("User %d sent: %s\n", obj.Message.PeerID, obj.Message.Text)

	//Ignoring messages in the group chat.
	if obj.Message.PeerID > 2000000000 {
		switch obj.Message.Text {
		case "+":
			sendMessage(EventStrings.SubscribeResponse, obj.Message.PeerID)
			updateUserState(ctx, obj.Message.FromID, obj.Message.PeerID, true)
		case "-":
			sendMessage(EventStrings.UnsubscribeResponse, obj.Message.PeerID)
			updateUserState(ctx, obj.Message.FromID, obj.Message.PeerID, false)
		default:
			sendMessage(EventStrings.DefaultResponse, obj.Message.PeerID)
		}
	}
}

func onMessageAllow(c context.Context, obj events.MessageAllowObject) {
	log.Printf("User %d allowed incoming messages with key %s\n", obj.UserID, obj.Key)
	query, err := DB.PrepareContext(c, fmt.Sprintf(`
	INSERT INTO %s (userID, subscribed) VALUES ($1, $2)
	ON CONFLICT DO NOTHING;
	`, TableName))
	if err != nil {
		log.Fatalf("Could not prepare db insert query:%v\n", err)
	}
	// Make user unsubscribed by default
	res, err := query.ExecContext(c, obj.UserID, false)
	if err != nil {
		log.Fatalf("Could not insert a value into db: %v\n", err)
	}
	defer query.Close()
	log.Printf("insert operation: %v\n", res)
}

func onMessageDeny(c context.Context, obj events.MessageDenyObject) {
	log.Printf("User %d denied incoming messages\n", obj.UserID)
	query, err := DB.PrepareContext(c, fmt.Sprintf("DELETE FROM %s WHERE userID = $1;", TableName))
	if err != nil {
		log.Fatalf("Could not prepare db delete query:%v\n", err)
	}
	res, err := query.ExecContext(c, obj.UserID)
	if err != nil {
		log.Fatalf("Could not insert a value into db: %v\n", err)
	}
	defer query.Close()
	log.Printf("DB delete operation: %v\n", res)
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}

	defer r.Body.Close()

	// Verify that the notification came from twitch using the secret.
	if !helix.VerifyEventSubNotification(SubSecret, r.Header, string(body)) {
		log.Println("No valid signature on subscription")
		return
	} else {
		log.Println("Verified signature for subscription")
	}

	var vals EventSubNotification
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&vals)
	if err != nil {
		log.Printf("Invalid webhook json: %v\n", err)
		return
	}

	// If there's a challenge in the request, respond with only the challenge to verify your eventsub.
	if vals.Challenge != "" {
		w.Write([]byte(vals.Challenge))
		return
	}

	var streamEvent helix.EventSubStreamOnlineEvent
	err = json.NewDecoder(bytes.NewReader(vals.Event)).Decode(&streamEvent)
	if err != nil {
		log.Printf("Invalid webhook json: %v\n", err)
		return
	}

	log.Printf("Got stream start webhook: %s in now online!\n", streamEvent.BroadcasterUserName)
	startMessaging(r.Context(), fmt.Sprintf(EventStrings.Notification, streamEvent.BroadcasterUserName, streamEvent.BroadcasterUserLogin))

	w.WriteHeader(200)
	w.Write([]byte("ok"))
}
