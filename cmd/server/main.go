// cmd/server/main.go
package main

import (
	"fmt"
	"log"
	"secure-messenger/internal/server"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("     Secure Messenger Server")
	fmt.Println("     Автор: Ваше Имя")
	fmt.Println("     Курсовая работа")
	fmt.Println("========================================")

	s := server.NewServer(":8080")
	if err := s.Start(); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}
}
