package client

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "io"
)

// Encrypt шифрует текст с использованием ключа
func Encrypt(key []byte, text string) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    // Создаем IV
    ciphertext := make([]byte, aes.BlockSize+len(text))
    iv := ciphertext[:aes.BlockSize]
    if _, err := io.ReadFull(rand.Reader, iv); err != nil {
        return "", err
    }

    // Шифруем
    stream := cipher.NewCFBEncrypter(block, iv)
    stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(text))

    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt расшифровывает текст
func Decrypt(key []byte, cryptoText string) (string, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    if len(ciphertext) < aes.BlockSize {
        return "", fmt.Errorf("ciphertext слишком короткий")
    }

    iv := ciphertext[:aes.BlockSize]
    ciphertext = ciphertext[aes.BlockSize:]

    stream := cipher.NewCFBDecrypter(block, iv)
    stream.XORKeyStream(ciphertext, ciphertext)

    return string(ciphertext), nil
}

// GenerateKeyFromPassword создает ключ из пароля
func GenerateKeyFromPassword(password string, salt []byte) []byte {
    hash := sha256.Sum256([]byte(password + string(salt)))
    return hash[:]
}
