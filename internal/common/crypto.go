package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

// GenerateSessionToken - генерирует токен сессии
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HashPassword - безопасное хэширование пароля с солью
func HashPassword(password, salt string) string {
	hash := sha256.New()
	hash.Write([]byte(password + salt))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// GenerateSalt - генерирует соль для пароля
func GenerateSalt() (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// AESEncrypt - шифрование AES-GCM
func AESEncrypt(plaintext, key string) (string, string, string, error) {
	// Создаем ключ из хэша
	keyHash := sha256.Sum256([]byte(key))
	aesKey := keyHash[:]

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", "", "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Разделяем nonce и ciphertext
	n := gcm.NonceSize()
	return base64.StdEncoding.EncodeToString(ciphertext[n:]),
		base64.StdEncoding.EncodeToString(nonce),
		"", nil
}

// AESDecrypt - расшифрование AES-GCM
func AESDecrypt(ciphertext, nonce, key string) (string, error) {
	keyHash := sha256.Sum256([]byte(key))
	aesKey := keyHash[:]

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := gcm.Open(nil, nonceBytes, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// SimpleEncrypt - для демонстрации (совместимость)
func SimpleEncrypt(text, recipient string) (encrypted, iv, tag string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	ivBytes := make([]byte, 12)
	rand.Read(ivBytes)

	return "🔐 " + encoded + " [для: " + recipient + "]",
		base64.StdEncoding.EncodeToString(ivBytes),
		"demo_tag"
}

// SimpleDecrypt - для демонстрации (совместимость)
func SimpleDecrypt(encrypted, iv, tag, recipient string) (string, error) {
	if !isEncrypted(encrypted) {
		return encrypted, nil
	}

	encoded := extractEncodedContent(encrypted)
	if encoded == "" {
		return encrypted, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded, nil
	}

	return string(decoded), nil
}

func isEncrypted(text string) bool {
	return len(text) > 3 && text[:3] == "🔐 "
}

func extractEncodedContent(encrypted string) string {
	if !isEncrypted(encrypted) {
		return ""
	}

	content := encrypted[3:] // Убираем "🔐 "
	parts := splitAt(content, "[для: ")
	if len(parts) > 0 {
		return parts[0]
	}
	return content
}

func splitAt(s, sep string) []string {
	for i := 0; i < len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}
