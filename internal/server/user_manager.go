package server

import (
	"errors"
	"secure-messenger/internal/common"
	"sync"
	"time"
)

// User представляет пользователя системы
type User struct {
	Username       string
	PasswordHash   string
	Salt           string
	IsOnline       bool
	LastSeen       time.Time
	JoinedAt       time.Time
	SessionToken   string
	SessionExpires time.Time
	PublicKey      string
	ConnectionID   string
}

// MessageHistory история сообщений
type MessageHistory struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IV        string    `json:"iv,omitempty"`
	AuthTag   string    `json:"auth_tag,omitempty"`
	Encrypted bool      `json:"encrypted"`
}

// UserManager управляет пользователями и их данными
type UserManager struct {
	users        map[string]*User
	messages     []MessageHistory
	sessions     map[string]*User // token -> user
	onlineUsers  map[string]bool  // username -> online status
	mu           sync.RWMutex
	messageLimit int
}

// NewUserManager создает новый менеджер пользователей
func NewUserManager() *UserManager {
	return &UserManager{
		users:        make(map[string]*User),
		messages:     make([]MessageHistory, 0),
		sessions:     make(map[string]*User),
		onlineUsers:  make(map[string]bool),
		messageLimit: 1000,
	}
}

// RegisterUser регистрирует нового пользователя
func (um *UserManager) RegisterUser(username, password string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	// Проверка существования пользователя
	if _, exists := um.users[username]; exists {
		return errors.New("пользователь уже существует")
	}

	// Валидация имени пользователя
	if !common.ValidateUsername(username) {
		return errors.New("недопустимое имя пользователя")
	}

	// Генерация соли и хэширование пароля
	salt, err := common.GenerateSalt()
	if err != nil {
		return errors.New("ошибка генерации соли")
	}

	passwordHash := common.HashPassword(password, salt)

	// Создание пользователя
	um.users[username] = &User{
		Username:     username,
		PasswordHash: passwordHash,
		Salt:         salt,
		IsOnline:     false,
		LastSeen:     time.Now(),
		JoinedAt:     time.Now(),
		PublicKey:    "",
	}

	return nil
}

// ValidateCredentials проверяет учетные данные
func (um *UserManager) ValidateCredentials(username, password string) (bool, error) {
	um.mu.RLock()
	user, exists := um.users[username]
	um.mu.RUnlock()

	if !exists {
		return false, errors.New("пользователь не найден")
	}

	// Проверка пароля
	passwordHash := common.HashPassword(password, user.Salt)
	return passwordHash == user.PasswordHash, nil
}

// CreateSession создает сессию для пользователя
func (um *UserManager) CreateSession(username string) string {
	um.mu.Lock()
	defer um.mu.Unlock()

	user, exists := um.users[username]
	if !exists {
		return ""
	}

	// Генерация токена сессии
	token, err := common.GenerateSessionToken()
	if err != nil {
		return ""
	}

	// Удаление старой сессии, если существует
	if user.SessionToken != "" {
		delete(um.sessions, user.SessionToken)
	}

	// Установка новой сессии
	user.SessionToken = token
	user.SessionExpires = time.Now().Add(24 * time.Hour)
	user.IsOnline = true
	user.LastSeen = time.Now()

	um.sessions[token] = user
	um.onlineUsers[username] = true

	return token
}

// ValidateSession проверяет валидность сессии
func (um *UserManager) ValidateSession(token string) (string, bool) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	user, exists := um.sessions[token]
	if !exists {
		return "", false
	}

	// Проверка срока действия сессии
	if time.Now().After(user.SessionExpires) {
		return "", false
	}

	return user.Username, true
}

// UpdateSession обновляет время сессии
func (um *UserManager) UpdateSession(username string) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if user, exists := um.users[username]; exists {
		user.SessionExpires = time.Now().Add(24 * time.Hour)
		user.LastSeen = time.Now()
	}
}

// Logout завершает сессию
func (um *UserManager) Logout(token string) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if user, exists := um.sessions[token]; exists {
		user.SessionToken = ""
		user.IsOnline = false
		delete(um.sessions, token)
		delete(um.onlineUsers, user.Username)
	}
}

// GetUser возвращает пользователя по имени
func (um *UserManager) GetUser(username string) (*User, bool) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	user, exists := um.users[username]
	return user, exists
}

// GetAllUsers возвращает всех пользователей
func (um *UserManager) GetAllUsers() []common.UserInfo {
	um.mu.RLock()
	defer um.mu.RUnlock()

	users := make([]common.UserInfo, 0, len(um.users))
	for _, user := range um.users {
		users = append(users, common.UserInfo{
			Username:  user.Username,
			PublicKey: user.PublicKey,
			IsOnline:  user.IsOnline,
			LastSeen:  user.LastSeen,
			JoinedAt:  user.JoinedAt,
		})
	}

	return users
}

// GetOnlineUsers возвращает онлайн пользователей
func (um *UserManager) GetOnlineUsers() []string {
	um.mu.RLock()
	defer um.mu.RUnlock()

	online := make([]string, 0, len(um.onlineUsers))
	for username := range um.onlineUsers {
		online = append(online, username)
	}
	return online
}

// SetOnline устанавливает статус онлайн
func (um *UserManager) SetOnline(username string, online bool) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if user, exists := um.users[username]; exists {
		user.IsOnline = online
		user.LastSeen = time.Now()

		if online {
			um.onlineUsers[username] = true
		} else {
			delete(um.onlineUsers, username)
		}
	}
}

// UpdatePublicKey обновляет публичный ключ
func (um *UserManager) UpdatePublicKey(username, publicKey string) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if user, exists := um.users[username]; exists {
		user.PublicKey = publicKey
	}
}

// AddMessage добавляет сообщение в историю
func (um *UserManager) AddMessage(msg common.Message) {
	um.mu.Lock()
	defer um.mu.Unlock()

	historyMsg := MessageHistory{
		ID:        msg.ID,
		Type:      msg.Type,
		Sender:    msg.Sender,
		Recipient: msg.Recipient,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
		IV:        msg.IV,
		AuthTag:   msg.AuthTag,
		Encrypted: msg.IV != "" || msg.AuthTag != "",
	}

	um.messages = append(um.messages, historyMsg)

	// Ограничение истории сообщений
	if len(um.messages) > um.messageLimit {
		um.messages = um.messages[1:]
	}
}

// GetUserHistory возвращает историю сообщений пользователя
func (um *UserManager) GetUserHistory(username string) []MessageHistory {
	um.mu.RLock()
	defer um.mu.RUnlock()

	var history []MessageHistory
	for _, msg := range um.messages {
		if msg.Recipient == "all" ||
			msg.Recipient == username ||
			msg.Sender == username {
			history = append(history, msg)
		}
	}

	return history
}

// GetConversationHistory возвращает историю диалога между двумя пользователями
func (um *UserManager) GetConversationHistory(user1, user2 string) []MessageHistory {
	um.mu.RLock()
	defer um.mu.RUnlock()

	var history []MessageHistory
	for _, msg := range um.messages {
		if (msg.Sender == user1 && msg.Recipient == user2) ||
			(msg.Sender == user2 && msg.Recipient == user1) {
			history = append(history, msg)
		}
	}

	return history
}

// GetOnlineCount возвращает количество онлайн пользователей
func (um *UserManager) GetOnlineCount() int {
	um.mu.RLock()
	defer um.mu.RUnlock()

	return len(um.onlineUsers)
}

// CleanupSessions очищает просроченные сессии
func (um *UserManager) CleanupSessions() {
	um.mu.Lock()
	defer um.mu.Unlock()

	now := time.Now()
	for token, user := range um.sessions {
		if now.After(user.SessionExpires) {
			user.SessionToken = ""
			user.IsOnline = false
			delete(um.sessions, token)
			delete(um.onlineUsers, user.Username)
		}
	}
}

// GetStatistics возвращает статистику системы
func (um *UserManager) GetStatistics() map[string]interface{} {
	um.mu.RLock()
	defer um.mu.RUnlock()

	return map[string]interface{}{
		"total_users":    len(um.users),
		"online_users":   len(um.onlineUsers),
		"total_messages": len(um.messages),
		"message_limit":  um.messageLimit,
	}
}
