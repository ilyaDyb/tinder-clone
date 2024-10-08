package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/ilyaDyb/go_rest_api/config"
	"github.com/ilyaDyb/go_rest_api/logger"

	"github.com/sirupsen/logrus"
)

const TypeReadMessages = "messages:reader"

type ReadMessagesPayload struct {
	ChatID uint
	UserID uint // this should be the user_id from which the message was sent
}

func NewReadMessagesTask(chatID, userID uint) (*asynq.Task, error) {
	payload, err := json.Marshal(ReadMessagesPayload{ChatID: chatID, UserID: userID})
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"service": "asynq",
		}).Errorf("Failed to start ReadMessagesTask with error: %v", err)
		return nil, err
	}
	return asynq.NewTask(TypeReadMessages, payload), nil
}

func HandleReadMessagesTask(ctx context.Context, t *asynq.Task) error {
	var p ReadMessagesPayload
	log.Println("HandleReadMessagesTask Was started")
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
        logger.Log.WithFields(logrus.Fields{
			"service": "asynq",
		}).Errorf("Failed to unmarchal data with error: %v", err)
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	query := `
		UPDATE messages SET is_read = true
		WHERE chat_id = ? AND sender_id = ?
	`
	log.Println(p.ChatID, p.UserID)
	if err := config.DB.Exec(query, p.ChatID, p.UserID).Error; err != nil {
		return err
	}
	
	return nil
}