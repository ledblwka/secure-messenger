// internal/server/client_handler.go
package server

import (
	"log"
	"net"
	"secure-messenger/internal/common"
	"time"
)

type ClientHandler struct {
	conn          net.Conn
	server        *Server
	username      string
	authenticated bool
	token         string
	lastActivity  time.Time
	done          chan bool
}

func NewClientHandler(conn net.Conn, server *Server) *ClientHandler {
	return &ClientHandler{
		conn:         conn,
		server:       server,
		lastActivity: time.Now(),
		done:         make(chan bool),
	}
}

func (ch *ClientHandler) Handle() {
	defer ch.conn.Close()

	log.Printf("Новое соединение от %s", ch.conn.RemoteAddr())

	for {
		select {
		case <-ch.done:
			return
		default:
			// Устанавливаем таймаут для чтения
			ch.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			msg, err := common.ReadMessage(ch.conn)
			if err != nil {
				log.Printf("Ошибка чтения от клиента: %v", err)
				ch.disconnect()
				return
			}

			ch.lastActivity = time.Now()
			ch.handleMessage(msg)
		}
	}
}

func (ch *ClientHandler) handleMessage(msg *common.Message) {
	switch msg.Type {
	case common.MsgLogin:
		ch.handleLogin(msg)
	case common.MsgRegister:
		ch.handleRegister(msg)
	case common.MsgLogout:
		ch.handleLogout()
	case common.MsgText, common.MsgEncrypted:
		ch.handleTextMessage(msg)
	case common.MsgPing:
		ch.handlePing()
	case common.MsgTyping:
		ch.handleTyping(msg)
	default:
		log.Printf("Неизвестный тип сообщения: %s", msg.Type)
	}
}

func (ch *ClientHandler) handleLogin(msg *common.Message) {
	username := msg.Sender

	// Простая проверка - принимаем любой логин для теста
	if username == "" {
		ch.SendMessage(&common.Message{
			Type:    common.MsgError,
			Content: "Имя пользователя не может быть пустым",
		})
		return
	}

	// Проверяем, не онлайн ли уже пользователь
	if _, exists := ch.server.GetClient(username); exists {
		ch.SendMessage(&common.Message{
			Type:    common.MsgError,
			Content: "Пользователь уже онлайн",
		})
		return
	}

	// Аутентификация успешна
	ch.username = username
	ch.authenticated = true
	ch.token = common.GenerateSessionToken(username)

	// Регистрируем клиента на сервере
	ch.server.RegisterClient(username, ch)

	// Отправляем успешный ответ
	ch.SendMessage(&common.Message{
		Type:      common.MsgSuccess,
		Content:   "Вход выполнен успешно",
		Token:     ch.token,
		Timestamp: time.Now(),
	})

	// Уведомляем всех о новом пользователе
	ch.server.BroadcastMessage(&common.Message{
		Type:      common.MsgUserJoined,
		Sender:    "Сервер",
		Recipient: "all",
		Content:   username + " присоединился к чату",
		Timestamp: time.Now(),
	}, username)

	log.Printf("Пользователь %s вошел в систему", username)
}

func (ch *ClientHandler) handleRegister(msg *common.Message) {
	username := msg.Sender

	if username == "" {
		ch.SendMessage(&common.Message{
			Type:      common.MsgError,
			Content:   "Имя пользователя не может быть пустым",
			Timestamp: time.Now(),
		})
		return
	}

	// Простая регистрация - проверяем, не существует ли пользователь
	if _, exists := ch.server.GetClient(username); exists {
		ch.SendMessage(&common.Message{
			Type:      common.MsgError,
			Content:   "Пользователь уже существует",
			Timestamp: time.Now(),
		})
		return
	}

	ch.SendMessage(&common.Message{
		Type:      common.MsgSuccess,
		Content:   "Регистрация выполнена успешно",
		Timestamp: time.Now(),
	})

	log.Printf("Зарегистрирован новый пользователь: %s", username)
}

func (ch *ClientHandler) handleLogout() {
	if ch.authenticated && ch.username != "" {
		ch.server.UnregisterClient(ch.username)

		// Уведомляем всех о выходе пользователя
		ch.server.BroadcastMessage(&common.Message{
			Type:      common.MsgUserLeft,
			Sender:    "Сервер",
			Recipient: "all",
			Content:   ch.username + " покинул чат",
			Timestamp: time.Now(),
		}, ch.username)

		log.Printf("Пользователь %s вышел из системы", ch.username)
	}
	ch.disconnect()
}

func (ch *ClientHandler) handleTextMessage(msg *common.Message) {
	if !ch.authenticated {
		ch.SendMessage(&common.Message{
			Type:      common.MsgError,
			Content:   "Требуется аутентификация",
			Timestamp: time.Now(),
		})
		return
	}

	// Устанавливаем отправителя и время
	msg.Sender = ch.username
	msg.Timestamp = time.Now()

	if msg.Recipient == "all" {
		// Рассылаем всем
		ch.server.BroadcastMessage(msg, ch.username)
	} else {
		// Отправляем конкретному пользователю
		if !ch.server.SendToUser(msg.Recipient, msg) {
			ch.SendMessage(&common.Message{
				Type:      common.MsgError,
				Content:   "Пользователь не найден или оффлайн",
				Timestamp: time.Now(),
			})
		}
	}

	log.Printf("Сообщение: %s -> %s: %s", ch.username, msg.Recipient, msg.Content)
}

func (ch *ClientHandler) handlePing() {
	ch.SendMessage(&common.Message{
		Type:      common.MsgPong,
		Timestamp: time.Now(),
	})
}

func (ch *ClientHandler) handleTyping(msg *common.Message) {
	if !ch.authenticated {
		return
	}

	// Пересылаем уведомление о наборе текста получателю
	if msg.Recipient != "all" && msg.Recipient != "" {
		ch.server.SendToUser(msg.Recipient, &common.Message{
			Type:      common.MsgTyping,
			Sender:    ch.username,
			Recipient: msg.Recipient,
			Content:   msg.Content,
			Timestamp: time.Now(),
		})
	}
}

func (ch *ClientHandler) SendMessage(msg *common.Message) {
	err := common.WriteMessage(ch.conn, msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения клиенту %s: %v", ch.username, err)
		ch.disconnect()
	}
}

func (ch *ClientHandler) Disconnect() {
	ch.done <- true
	ch.conn.Close()

	if ch.authenticated && ch.username != "" {
		ch.server.UnregisterClient(ch.username)
	}
}

func (ch *ClientHandler) disconnect() {
	close(ch.done)
	ch.conn.Close()

	if ch.authenticated && ch.username != "" {
		ch.server.UnregisterClient(ch.username)
	}
}

func (ch *ClientHandler) LastActivity() time.Time {
	return ch.lastActivity
}

func (ch *ClientHandler) GetUsername() string {
	return ch.username
}
