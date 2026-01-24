package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"
)

// GenerateSessionToken генерирует безопасный токен сессии
func GenerateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// HashPassword создает хэш пароля с солью
func HashPassword(password, salt string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	hash.Write([]byte(salt))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// GenerateSalt генерирует случайную соль
func GenerateSalt() (string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(saltBytes), nil
}

// EncryptMessage шифрует сообщение
func EncryptMessage(text, key string) (string, string, error) {
	// Создаем ключ из хэша
	keyHash := sha256.Sum256([]byte(key))
	aesKey := keyHash[:]

	// Создаем AES cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", "", err
	}

	// Создаем GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	// Генерируем nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	// Шифруем сообщение
	ciphertext := gcm.Seal(nil, nonce, []byte(text), nil)

	// Кодируем в base64
	encryptedText := base64.StdEncoding.EncodeToString(ciphertext)
	nonceStr := base64.StdEncoding.EncodeToString(nonce)

	return encryptedText, nonceStr, nil
}

// DecryptMessage расшифровывает сообщение
func DecryptMessage(encryptedText, nonceStr, key string) (string, error) {
	// Создаем ключ из хэша
	keyHash := sha256.Sum256([]byte(key))
	aesKey := keyHash[:]

	// Декодируем из base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceStr)
	if err != nil {
		return "", err
	}

	// Создаем AES cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	// Создаем GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Расшифровываем сообщение
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// SimpleEncrypt простая функция шифрования для демонстрации
func SimpleEncrypt(text, recipient string) (encrypted, iv, tag string) {
	// Кодируем сообщение в base64
	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	// Генерируем IV
	ivBytes := make([]byte, 12)
	rand.Read(ivBytes)

	// Форматируем результат
	encrypted = "🔐 " + encoded + " [для: " + recipient + "]"
	iv = base64.StdEncoding.EncodeToString(ivBytes)
	tag = "demo_tag_" + recipient

	return encrypted, iv, tag
}

// SimpleDecrypt простая функция расшифрования для демонстрации
func SimpleDecrypt(encrypted, iv, tag, recipient string) (string, error) {
	// Проверяем, зашифровано ли сообщение
	if !strings.HasPrefix(encrypted, "🔐 ") {
		return encrypted, nil
	}

	// Извлекаем закодированный текст
	content := strings.TrimPrefix(encrypted, "🔐 ")
	parts := strings.Split(content, " [для: ")
	if len(parts) < 2 {
		return content, nil
	}

	encodedText := parts[0]

	// Декодируем из base64
	decoded, err := base64.StdEncoding.DecodeString(encodedText)
	if err != nil {
		return encodedText, nil
	}

	return string(decoded), nil
}

// GenerateMessageID генерирует уникальный ID для сообщения
func GenerateMessageID() (string, error) {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(idBytes), nil
}

// ValidateUsername проверяет валидность имени пользователя
func ValidateUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}

	// Разрешаем только буквы, цифры и подчеркивание
	for _, ch := range username {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_') {
			return false
		}
	}
	return true
}
