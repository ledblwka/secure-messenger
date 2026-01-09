// internal/client/ui.go
package client

import (
	"bufio"
	"fmt"
	"os"
	"secure-messenger/internal/common"
	"strings"
)

type ConsoleUI struct {
	client      *ChatClient
	scanner     *bufio.Scanner
	currentUser string
}

func NewConsoleUI() *ConsoleUI {
	return &ConsoleUI{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (ui *ConsoleUI) SetClient(client *ChatClient) {
	ui.client = client
}

func (ui *ConsoleUI) Run() {
	ui.showWelcome()
	ui.showConnectionMenu()
}

func (ui *ConsoleUI) showWelcome() {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     Secure Messenger v1.0            ║")
	fmt.Println("║     Шифрованный обмен сообщениями    ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
}

func (ui *ConsoleUI) showConnectionMenu() {
	fmt.Println("\n=== ПОДКЛЮЧЕНИЕ К СЕРВЕРУ ===")
	fmt.Print("Адрес сервера [localhost:8080]: ")
	ui.scanner.Scan()
	address := ui.scanner.Text()
	if address == "" {
		address = "localhost:8080"
	}

	ui.client = NewChatClient(address)
	err := ui.client.Connect()
	if err != nil {
		fmt.Printf("Ошибка подключения: %v\n", err)
		return
	}

	ui.client.SetUI(ui)
	fmt.Println("✓ Подключено к серверу")

	ui.showAuthMenu()
}

func (ui *ConsoleUI) showAuthMenu() {
	for {
		fmt.Println("\n=== АУТЕНТИФИКАЦИЯ ===")
		fmt.Println("1. Вход")
		fmt.Println("2. Регистрация")
		fmt.Println("3. Выход")
		fmt.Print("Выберите действие: ")

		ui.scanner.Scan()
		choice := ui.scanner.Text()

		switch choice {
		case "1":
			ui.login()
			if ui.currentUser != "" {
				ui.showChatMenu()
				return
			}
		case "2":
			ui.register()
		case "3":
			fmt.Println("Выход...")
			ui.client.Disconnect()
			os.Exit(0)
		default:
			fmt.Println("Неверный выбор")
		}
	}
}

func (ui *ConsoleUI) login() {
	fmt.Print("Имя пользователя: ")
	ui.scanner.Scan()
	username := strings.TrimSpace(ui.scanner.Text())

	if username == "" {
		fmt.Println("Имя пользователя не может быть пустым")
		return
	}

	// Для теста - пароль не важен
	password := "test"

	err := ui.client.Login(username, password)
	if err != nil {
		fmt.Printf("Ошибка входа: %v\n", err)
		return
	}

	fmt.Println("✓ Вход выполнен успешно!")
	ui.currentUser = username
}

func (ui *ConsoleUI) register() {
	fmt.Print("Имя пользователя: ")
	ui.scanner.Scan()
	username := strings.TrimSpace(ui.scanner.Text())

	if username == "" {
		fmt.Println("Имя пользователя не может быть пустым")
		return
	}

	// Простая регистрация
	err := ui.client.Register(username, "password")
	if err != nil {
		fmt.Printf("Ошибка регистрации: %v\n", err)
		return
	}

	fmt.Println("✓ Регистрация выполнена успешно!")
}

func (ui *ConsoleUI) showChatMenu() {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Printf("Добро пожаловать, %s!\n", ui.currentUser)
	fmt.Println("Формат сообщения: @username текст")
	fmt.Println("Для всех: @all текст")
	fmt.Println("Введите 'exit' для выхода")
	fmt.Println(strings.Repeat("=", 40))

	for {
		fmt.Print("> ")
		ui.scanner.Scan()
		input := strings.TrimSpace(ui.scanner.Text())

		if input == "" {
			continue
		}

		if strings.ToLower(input) == "exit" {
			fmt.Println("Выход...")
			ui.client.Disconnect()
			os.Exit(0)
		}

		// Парсим ввод
		parts := strings.SplitN(input, " ", 2)
		if len(parts) < 2 {
			fmt.Println("❌ Формат: @получатель текст")
			continue
		}

		recipient := parts[0]
		content := parts[1]

		// Убираем @ если есть
		if strings.HasPrefix(recipient, "@") {
			recipient = recipient[1:]
		}

		// Отправляем сообщение
		ui.client.SendMessage(recipient, content, false)

		// Показываем свое сообщение
		fmt.Printf("Вы -> %s: %s\n", recipient, content)
	}
}

// Реализация интерфейса UIHandler
func (ui *ConsoleUI) ShowMessage(msg *common.Message) {
	if msg.Sender != ui.currentUser { // Не показываем свои же сообщения повторно
		fmt.Printf("[%s] %s: %s\n",
			msg.Timestamp.Format("15:04:05"),
			msg.Sender,
			msg.Content)
	}
}

func (ui *ConsoleUI) UpdateUserList(users []string) {
	// Пока не реализовано
}

func (ui *ConsoleUI) ShowError(err string) {
	fmt.Printf("❌ Ошибка: %s\n", err)
}
