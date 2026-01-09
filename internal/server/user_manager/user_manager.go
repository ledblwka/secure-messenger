package user_manager

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Salt         string `json:"salt"`
	CreatedAt    string `json:"created_at"`
}

type UserManager struct {
	users map[string]*User
	mu    sync.RWMutex
	file  string
}

func NewUserManager() *UserManager {
	um := &UserManager{
		users: make(map[string]*User),
		file:  "users.json",
	}
	um.loadUsers()
	return um
}

func (um *UserManager) loadUsers() {
	um.mu.Lock()
	defer um.mu.Unlock()

	data, err := os.ReadFile(um.file)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, создаем пустой
			um.saveUsers()
		}
		return
	}

	json.Unmarshal(data, &um.users)
}

func (um *UserManager) saveUsers() {
	data, err := json.MarshalIndent(um.users, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(um.file, data, 0644)
}

func (um *UserManager) hashPassword(password, salt string) string {
	hash := sha256.New()
	hash.Write([]byte(password + salt))
	return hex.EncodeToString(hash.Sum(nil))
}

func (um *UserManager) Register(username, password string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	if _, exists := um.users[username]; exists {
		return errors.New("пользователь уже существует")
	}

	if len(password) < 6 {
		return errors.New("пароль должен содержать минимум 6 символов")
	}

	// Генерируем соль и хэшируем пароль
	salt := generateSalt()
	passwordHash := um.hashPassword(password, salt)

	um.users[username] = &User{
		Username:     username,
		PasswordHash: passwordHash,
		Salt:         salt,
		CreatedAt:    time.Now().Format(time.RFC3339),
	}

	um.saveUsers()
	return nil
}

func (um *UserManager) Authenticate(username, password string) (*User, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	user, exists := um.users[username]
	if !exists {
		return nil, errors.New("пользователь не найден")
	}

	// Проверяем пароль
	expectedHash := um.hashPassword(password, user.Salt)
	if expectedHash != user.PasswordHash {
		return nil, errors.New("неверный пароль")
	}

	return user, nil
}

func (um *UserManager) UserExists(username string) bool {
	um.mu.RLock()
	defer um.mu.RUnlock()

	_, exists := um.users[username]
	return exists
}

func generateSalt() string {
	// Генерируем случайную строку для соли
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
