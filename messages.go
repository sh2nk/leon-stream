package main

import (
	"context"
	"fmt"
	"log"

	"github.com/SevereCloud/vksdk/v2/api/params"
)

func sendMessage(messageText string, peerID int) {
	builder := params.NewMessagesSendBuilder()
	builder.Message(messageText)
	builder.RandomID(int(randomInt32()))
	builder.PeerID(peerID)

	_, err := VK.MessagesSend(builder.Params)
	if err != nil {
		log.Fatal(err)
	}
}

func startMessaging(ctx context.Context, msg string) {
	peers := make([]int, 0)
	res, err := DB.QueryContext(ctx, fmt.Sprintf("SELECT peerID FROM %s WHERE subscribed = true", TableName))
	if err != nil {
		log.Fatalf("Fatal error during mass messaging: %v\n", err)
	}
	defer res.Close()
	for res.Next() {
		var peer int
		err = res.Scan(&peer)
		if err != nil {
			log.Fatalf("Fatal error during mass messaging: %v\n", err)
		}
		peers = append(peers, peer)
	}

	for _, peer := range peers {
		sendMessage(msg, peer)
	}
}
