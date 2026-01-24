package common

import "time"

// Типы сообщений
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
	MsgPing       = "ping"
	MsgPong       = "pong"
)

// Message структура сообщения
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

// UserInfo информация о пользователе
type UserInfo struct {
	Username  string    `json:"username"`
	PublicKey string    `json:"public_key,omitempty"`
	IsOnline  bool      `json:"is_online"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
	JoinedAt  time.Time `json:"joined_at,omitempty"`
}

// AuthRequest запрос аутентификации
type AuthRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	PublicKey    string `json:"public_key,omitempty"`
}

// AuthResponse ответ аутентификации
type AuthResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	Username     string `json:"username,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	Error        string `json:"error,omitempty"`
}

// RegisterRequest запрос регистрации
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ChatMessage сообщение в чате
type ChatMessage struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Content   string    `json:"content"`
	Encrypted bool      `json:"encrypted"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
}
