// internal/common/message.go
package common

import (
	"time"
)

type MessageType string

const (
	MsgLogin      MessageType = "LOGIN"
	MsgRegister   MessageType = "REGISTER"
	MsgText       MessageType = "TEXT"
	MsgEncrypted  MessageType = "ENCRYPTED"
	MsgError      MessageType = "ERROR"
	MsgSuccess    MessageType = "SUCCESS"
	MsgLogout     MessageType = "LOGOUT"
	MsgPing       MessageType = "PING"
	MsgPong       MessageType = "PONG"
	MsgTyping     MessageType = "TYPING"
	MsgUserList   MessageType = "USER_LIST"
	MsgUserJoined MessageType = "USER_JOINED"
	MsgUserLeft   MessageType = "USER_LEFT"
)

type Message struct {
	Type      MessageType `json:"type"`
	Sender    string      `json:"sender,omitempty"`
	Recipient string      `json:"recipient,omitempty"`
	Content   string      `json:"content,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Token     string      `json:"token,omitempty"`
	Data      []byte      `json:"data,omitempty"`
}
