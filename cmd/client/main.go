// cmd/client/main.go
package main

import (
	"secure-messenger/internal/client"
)

func main() {
	ui := client.NewConsoleUI()
	ui.Run()
}
