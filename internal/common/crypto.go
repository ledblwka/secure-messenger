// internal/common/crypto.go
package common

import (
	"crypto/rand"
	"encoding/base64"
)

// Заглушки для функций шифрования
func GenerateSessionToken(username string) string {
	return "token-" + username + "-" + randomString(16)
}

func GenerateKeyFromPassword(password string, salt []byte) []byte {
	// Простая реализация - в реальном проекте используйте PBKDF2
	key := make([]byte, 32)
	copy(key, []byte(password))
	return key
}

func EncryptString(key []byte, text string) (string, string, error) {
	// Простая заглушка
	encrypted := base64.StdEncoding.EncodeToString([]byte(text))
	return encrypted, "iv-placeholder", nil
}

func DecryptString(key []byte, encryptedText, iv string) (string, error) {
	// Простая заглушка
	decoded, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[b[i]%byte(len(letters))]
	}
	return string(b)
}
