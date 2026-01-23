package server

import (
	"errors"
	"secure-messenger/internal/common"
	"sync"
	"time"
)

type User struct {
	Username       string
	PasswordHash   string
	Salt           string
	IsOnline       bool
	LastSeen       time.Time
	SessionToken   string
	SessionExpires time.Time
	PublicKey      string
}

type MessageHistory struct {
	Type      string    `json:"type"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IV        string    `json:"iv,omitempty"`
	AuthTag   string    `json:"auth_tag,omitempty"`
	KeyID     string    `json:"key_id,omitempty"`
}

type UserManager struct {
	users    map[string]*User
	messages []MessageHistory
	sessions map[string]*User // token -> user
	mu       sync.RWMutex
}

func NewUserManager() *UserManager {
	return &UserManager{
		users:    make(map[string]*User),
		messages: make([]MessageHistory, 0),
		sessions: make(map[string]*User),
	}
}

// RegisterUser регистрирует нового пользователя
func (um *UserManager) RegisterUser(username, password string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if _, exists := um.users[username]; exists {
		return errors.New("пользователь уже существует")
	}

	// Генерируем соль и хэшируем пароль
	salt, _ := common.GenerateSalt()
	passwordHash := common.HashPassword(password, salt)

	um.users[username] = &User{
		Username:     username,
		PasswordHash: passwordHash,
		Salt:         salt,
		IsOnline:     false,
		LastSeen:     time.Now(),
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

	// Хэшируем пароль с солью пользователя
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

	// Удаляем старую сессию, если есть
	if user.SessionToken != "" {
		delete(um.sessions, user.SessionToken)
	}

	// Генерируем новый токен
	token, _ := common.GenerateSessionToken()
	user.SessionToken = token
	user.SessionExpires = time.Now().Add(24 * time.Hour)
	user.IsOnline = true
	user.LastSeen = time.Now()

	um.sessions[token] = user
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

	if time.Now().After(user.SessionExpires) {
		delete(um.sessions, token)
		user.SessionToken = ""
		user.IsOnline = false
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
		})
	}

	return users
}

// SetOnline устанавливает статус онлайн
func (um *UserManager) SetOnline(username string, online bool) {
	um.mu.Lock()
	defer um.mu.Unlock()

	if user, exists := um.users[username]; exists {
		user.IsOnline = online
		user.LastSeen = time.Now()
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
		Type:      msg.Type,
		Sender:    msg.Sender,
		Recipient: msg.Recipient,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
		IV:        msg.IV,
		AuthTag:   msg.AuthTag,
		KeyID:     msg.KeyID,
	}

	um.messages = append(um.messages, historyMsg)

	// Ограничиваем историю 1000 сообщений
	if len(um.messages) > 1000 {
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

// GetOnlineCount возвращает количество онлайн пользователей
func (um *UserManager) GetOnlineCount() int {
	um.mu.RLock()
	defer um.mu.RUnlock()

	count := 0
	for _, user := range um.users {
		if user.IsOnline {
			count++
		}
	}
	return count
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
		}
	}
}
