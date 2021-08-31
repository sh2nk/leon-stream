package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/SevereCloud/vksdk/v2/api/params"
)

func checkDB(ctx context.Context) (bool, bool, error) {
	var exists bool
	empty := true
	res, err := DB.QueryContext(ctx, "select EXISTS (select * from information_schema.tables where TABLE_NAME = $1 and table_schema = 'public') as table_exists;", TableName)
	if err != nil {
		return false, false, err
	}
	defer res.Close()
	if res.Next() {
		err = res.Scan(&exists)
		if err != nil {
			return false, false, err
		}
	}
	if exists {
		res, err := DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT CASE 
			WHEN EXISTS (SELECT * FROM %s LIMIT 1) THEN 0
			ELSE 1 
		END
	    `, TableName))
		if err != nil {
			return false, false, err
		}
		defer res.Close()
		if res.Next() {
			err = res.Scan(&empty)
			if err != nil {
				return false, false, err
			}
		}
	}
	return exists, empty, nil
}

func syncDB(ctx context.Context, groupID int) error {
	log.Println("Starting db sync...")

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return nil
	}

	builder := params.NewGroupsGetMembersBuilder()
	builder.GroupID(strconv.Itoa(groupID))
	builder.Sort("id_asc")

	users, err := VK.GroupsGetMembers(builder.Params)
	if err != nil {
		return err
	}

	for i := 0; i < users.Count; i++ {
		builder := params.NewMessagesIsMessagesFromGroupAllowedBuilder()
		builder.GroupID(groupID)
		builder.UserID(users.Items[i])

		res, err := VK.MessagesIsMessagesFromGroupAllowed(builder.Params)
		if err != nil {
			return err
		}

		if res.IsAllowed {
			_, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (userID, subscribed) VALUES ($1, $2);", TableName), users.Items[i], false)
			if err != nil {
				return err
			}
			log.Printf("Found user %d with allowed messages, added to db\n", users.Items[i])
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func createDB(ctx context.Context) {
	query, err := DB.PrepareContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (userID integer primary key, subscribed boolean, peerID integer);", TableName))
	if err != nil {
		log.Fatalf("Could not prepare db create query:%v\n", err)
	}
	_, err = query.ExecContext(ctx)
	if err != nil {
		log.Fatalf("Unable to create database: %v\n", err)
	}
	defer query.Close()
}

func updateUserState(ctx context.Context, userID int, peerID int, newState bool) {
	_, err := DB.ExecContext(ctx, fmt.Sprintf(`
	INSERT INTO %s (userID, subscribed, peerID) VALUES ($1, $2, $3)
	ON CONFLICT (userID)
	DO UPDATE SET subscribed = $2, peerID = $3;
	`, TableName), userID, newState, peerID)
	if err != nil {
		log.Fatalf("Falied to update user state: %v\n", err)
	}
}
