package utils

import (
	"github.com/ilyaDyb/go_rest_api/models"
	"github.com/rosberry/go-pagination"
)

// MessageResponse represents a generic message response
type MessageResponse struct {
    Message string `json:"message"`
}

// ErrorResponse represents a generic error response
type ErrorResponse struct {
    Error string `json:"error"`
}

type ModelResponse struct {
    Model string `json:"Model fields"`
}

type UsersListResponse struct {
	Result     bool                 `json:"result"`
	Users      []models.User        `json:"users"`
	Pagination *pagination.PageInfo `json:"pagination"`
}

type ChatsListResponse struct { 
	ChatID uint `json:"chat_id"`
	Username string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname string `json:"lastname"`
	PhotoURL string `json:"photo_url"`
	LastMessage string `json:"last_message"`
	IsRead bool `json:"is_read"`
	SenderUsername string `json:"sender_username"`
}