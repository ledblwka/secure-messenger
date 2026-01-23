package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"secure-messenger/internal/common"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketServer struct {
	userManager *UserManager
	clients     map[string]*websocket.Conn
	mu          sync.RWMutex
}

func NewWebSocketServer(userManager *UserManager) *WebSocketServer {
	return &WebSocketServer{
		userManager: userManager,
		clients:     make(map[string]*websocket.Conn),
	}
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// ✅ ИСПРАВЛЕНО: Не блокируем соединение
	// Вместо чтения в основном потоке, запускаем горутину
	go s.handleConnection(conn)
}

func (s *WebSocketServer) handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	// Устанавливаем таймаут для аутентификации
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	var authMsg common.Message
	if err := conn.ReadJSON(&authMsg); err != nil {
		log.Println("Auth message error:", err)
		return
	}

	// Сбрасываем таймаут после успешной аутентификации
	conn.SetReadDeadline(time.Time{})

	// Проверяем аутентификацию
	username, valid := s.authenticate(authMsg)
	if !valid {
		sendError(conn, "Authentication failed")
		return
	}

	// Регистрируем клиент
	s.mu.Lock()
	// Закрываем старое соединение, если пользователь уже подключен
	if oldConn, exists := s.clients[username]; exists {
		oldConn.Close()
		delete(s.clients, username)
	}
	s.clients[username] = conn
	s.mu.Unlock()

	log.Printf("✅ User connected: %s", username)

	// Отправляем приветственное сообщение
	s.sendWelcomeMessage(username, conn)

	// Уведомляем всех о новом пользователе
	s.broadcastUserJoined(username)

	// Обновляем список пользователей
	s.sendUserListToAll()

	// Отправляем историю
	s.sendHistoryToUser(username, conn)

	// Обработка сообщений
	s.handleMessages(conn, username)
}

func (s *WebSocketServer) authenticate(msg common.Message) (string, bool) {
	if msg.Type != common.MsgAuth {
		return "", false
	}

	// Проверяем токен сессии
	username, valid := s.userManager.ValidateSession(msg.SessionToken)
	if !valid {
		return "", false
	}

	// Обновляем статус
	s.userManager.SetOnline(username, true)
	return username, true
}

func (s *WebSocketServer) handleMessages(conn *websocket.Conn, username string) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.clients, username)
		s.mu.Unlock()

		s.userManager.SetOnline(username, false)
		s.broadcastUserLeft(username)
		s.sendUserListToAll()

		log.Printf("❌ User disconnected: %s", username)
	}()

	for {
		var msg common.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		msg.Sender = username
		msg.Timestamp = time.Now()

		switch msg.Type {
		case common.MsgGeneral:
			s.handleGeneralMessage(msg)
		case common.MsgPrivate:
			s.handlePrivateMessage(msg)
		case common.MsgTyping:
			s.handleTypingNotification(msg)
		}
	}
}

func (s *WebSocketServer) handleGeneralMessage(msg common.Message) {
	if msg.Recipient == "" {
		msg.Recipient = "all"
	}

	s.userManager.AddMessage(msg)
	s.broadcastToAllExcept(msg, msg.Sender)
}

func (s *WebSocketServer) handlePrivateMessage(msg common.Message) {
	if msg.Recipient != "" && msg.Recipient != "all" && msg.Recipient != msg.Sender {
		s.userManager.AddMessage(msg)
		s.sendToUser(msg.Recipient, msg)
	}
}

func (s *WebSocketServer) handleTypingNotification(msg common.Message) {
	msg.Type = common.MsgTyping
	if msg.Recipient != "" && msg.Recipient != "all" {
		s.sendToUser(msg.Recipient, msg)
	}
}

func (s *WebSocketServer) sendWelcomeMessage(username string, conn *websocket.Conn) {
	welcomeMsg := common.Message{
		Type:    common.MsgSuccess,
		Content: "Добро пожаловать в Secure Messenger!",
	}
	conn.WriteJSON(welcomeMsg)
}

func (s *WebSocketServer) broadcastUserJoined(username string) {
	msg := common.Message{
		Type:      common.MsgUserJoined,
		Sender:    username,
		Content:   "присоединился(ась) к чату",
		Timestamp: time.Now(),
	}

	s.broadcastToAll(msg)
}

func (s *WebSocketServer) broadcastUserLeft(username string) {
	msg := common.Message{
		Type:      common.MsgUserLeft,
		Sender:    username,
		Content:   "покинул(а) чат",
		Timestamp: time.Now(),
	}

	s.broadcastToAll(msg)
}

func (s *WebSocketServer) broadcastToAll(msg common.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.clients {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *WebSocketServer) broadcastToAllExcept(msg common.Message, except string) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for username, conn := range s.clients {
		if username != except {
			conn.WriteMessage(websocket.TextMessage, data)
		}
	}
}

func (s *WebSocketServer) sendToUser(recipient string, msg common.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	s.mu.RLock()
	conn, exists := s.clients[recipient]
	s.mu.RUnlock()

	if exists && conn != nil {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *WebSocketServer) sendUserListToAll() {
	users := s.userManager.GetAllUsers()

	msg := common.Message{
		Type:  common.MsgUsersList,
		Users: users,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.clients {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *WebSocketServer) sendHistoryToUser(username string, conn *websocket.Conn) {
	history := s.userManager.GetUserHistory(username)

	for _, msg := range history {
		historyMsg := common.Message{
			Type:      common.MsgHistory,
			Sender:    msg.Sender,
			Recipient: msg.Recipient,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		}
		conn.WriteJSON(historyMsg)
	}
}

func sendError(conn *websocket.Conn, message string) {
	msg := common.Message{
		Type:    common.MsgError,
		Content: message,
	}
	conn.WriteJSON(msg)
}
