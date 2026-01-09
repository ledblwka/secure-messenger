// internal/common/protocol.go
package common

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"
)

func WriteMessage(conn net.Conn, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Добавляем разделитель
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}

func ReadMessage(conn net.Conn) (*Message, error) {
	// Устанавливаем таймаут для чтения
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	decoder := json.NewDecoder(conn)
	var msg Message
	err := decoder.Decode(&msg)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, err
	}
	return &msg, nil
}
