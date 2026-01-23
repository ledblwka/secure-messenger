package common

import (
	"time"
)

// Типы сообщений для WebSocket
const (
	MsgRegister   = "register"
	MsgLogin      = "login"
	MsgLogout     = "logout"
	MsgGeneral    = "general"
	MsgPrivate    = "private"
	MsgUsersList  = "users_list"
	MsgUserJoined = "user_joined"
	MsgUserLeft   = "user_left"
	MsgError      = "error"
	MsgSuccess    = "success"
	MsgTyping     = "typing"
	MsgAuth       = "auth"
	MsgHistory    = "history"
)

// Структура сообщения
type Message struct {
	Type         string     `json:"type"`
	ID           string     `json:"id,omitempty"`
	Sender       string     `json:"sender,omitempty"`
	Recipient    string     `json:"recipient,omitempty"`
	Content      string     `json:"content,omitempty"`
	Timestamp    time.Time  `json:"timestamp"`
	Users        []UserInfo `json:"users,omitempty"`
	Error        string     `json:"error,omitempty"`
	IV           string     `json:"iv,omitempty"`
	AuthTag      string     `json:"auth_tag,omitempty"`
	KeyID        string     `json:"key_id,omitempty"`
	SessionToken string     `json:"session_token,omitempty"`
	Username     string     `json:"username,omitempty"`
	Password     string     `json:"password,omitempty"`
}

// Информация о пользователе
type UserInfo struct {
	Username  string    `json:"username"`
	PublicKey string    `json:"public_key,omitempty"`
	IsOnline  bool      `json:"is_online"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
}

// Аутентификация
type AuthRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	PublicKey    string `json:"public_key,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
}

// Ответ аутентификации
type AuthResponse struct {
	Success      bool   `json:"success"`
	Username     string `json:"username,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	Error        string `json:"error,omitempty"`
}
