// internal/server/server.go
package server

import (
	"encoding/json"
	"log"
	"net"
	"secure-messenger/internal/common"
	"sync"
	"time"
)

type Server struct {
	address  string
	listener net.Listener
	clients  map[string]*ClientHandler
	mu       sync.RWMutex
	done     chan bool
}

func NewServer(address string) *Server {
	return &Server{
		address: address,
		clients: make(map[string]*ClientHandler),
		done:    make(chan bool),
	}
}

func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	log.Printf("Сервер запущен на %s", s.address)

	go s.acceptConnections()

	// Запускаем горутину для очистки неактивных клиентов
	go s.cleanupInactiveClients()

	return nil
}

func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.done:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("Ошибка при принятии соединения: %v", err)
				continue
			}

			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	handler := NewClientHandler(conn, s)
	handler.Handle()
}

func (s *Server) RegisterClient(username string, handler *ClientHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[username] = handler
	log.Printf("Клиент зарегистрирован: %s", username)

	// Рассылаем обновленный список пользователей
	s.broadcastUserList()
}

func (s *Server) UnregisterClient(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[username]; exists {
		delete(s.clients, username)
		log.Printf("Клиент удален: %s", username)

		// Рассылаем обновленный список пользователей
		s.broadcastUserList()
	}
}

func (s *Server) GetClient(username string) (*ClientHandler, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[username]
	return client, exists
}

func (s *Server) BroadcastMessage(msg *common.Message, excludeUsername string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for username, client := range s.clients {
		if username != excludeUsername {
			client.SendMessage(msg)
		}
	}
}

func (s *Server) SendToUser(username string, msg *common.Message) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if client, exists := s.clients[username]; exists {
		client.SendMessage(msg)
		return true
	}
	return false
}

func (s *Server) broadcastUserList() {
	users := s.GetOnlineUsers()

	// Используем поле Data вместо Content для списка пользователей
	data, _ := json.Marshal(users)

	msg := &common.Message{
		Type: common.MsgUserList,
		Data: data,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		client.SendMessage(msg)
	}
}

func (s *Server) GetOnlineUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]string, 0, len(s.clients))
	for username := range s.clients {
		users = append(users, username)
	}
	return users
}

func (s *Server) cleanupInactiveClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			for username, client := range s.clients {
				if time.Since(client.LastActivity()) > 5*time.Minute {
					log.Printf("Клиент %s неактивен, отключаем", username)
					client.Disconnect()
					delete(s.clients, username)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *Server) Stop() {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, client := range s.clients {
		client.Disconnect()
	}

	log.Println("Сервер остановлен")
}
