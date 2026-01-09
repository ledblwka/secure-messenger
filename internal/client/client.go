package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"secure-messenger/internal/common"
	"time"
)

type ChatClient struct {
	conn          net.Conn
	serverAddr    string
	username      string
	token         string
	encryptionKey []byte
	connected     bool
	done          chan bool
	messageChan   chan *common.Message
	ui            UIHandler
}

type UIHandler interface {
	ShowMessage(msg *common.Message)
	UpdateUserList(users []string)
	ShowError(err string)
	SetClient(client *ChatClient)
}

func NewChatClient(serverAddr string) *ChatClient {
	return &ChatClient{
		serverAddr:  serverAddr,
		done:        make(chan bool),
		messageChan: make(chan *common.Message, 100),
	}
}

func (c *ChatClient) Connect() error {
	var err error
	c.conn, err = net.Dial("tcp", c.serverAddr)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к серверу: %v", err)
	}

	c.connected = true
	log.Printf("Подключено к серверу %s", c.serverAddr)

	// Запускаем горутину для чтения сообщений
	go c.readMessages()

	// Запускаем горутину для обработки сообщений
	go c.processMessages()

	// Запускаем пинг
	go c.keepAlive()

	return nil
}

func (c *ChatClient) Login(username, password string) error {
	msg := &common.Message{
		Type:    common.MsgLogin,
		Sender:  username,
		Content: password,
	}

	err := common.WriteMessage(c.conn, msg)
	if err != nil {
		return err
	}

	// Ждем ответа
	response, err := common.ReadMessage(c.conn)
	if err != nil {
		return err
	}

	if response.Type == common.MsgSuccess {
		c.username = username
		c.token = response.Token
		return nil
	} else if response.Type == common.MsgError {
		return fmt.Errorf(response.Content)
	}

	return fmt.Errorf("неизвестный ответ от сервера")
}

func (c *ChatClient) Register(username, password string) error {
	msg := &common.Message{
		Type:    common.MsgRegister,
		Sender:  username,
		Content: password,
	}

	err := common.WriteMessage(c.conn, msg)
	if err != nil {
		return err
	}

	response, err := common.ReadMessage(c.conn)
	if err != nil {
		return err
	}

	if response.Type == common.MsgSuccess {
		return nil
	} else if response.Type == common.MsgError {
		return fmt.Errorf(response.Content)
	}

	return fmt.Errorf("неизвестный ответ от сервера")
}

func (c *ChatClient) SendMessage(recipient, content string, encrypted bool) {
	msgType := common.MsgText
	if encrypted && c.encryptionKey != nil {
		msgType = common.MsgEncrypted
		// Шифруем сообщение
		encryptedText, iv, err := common.EncryptString(c.encryptionKey, content)
		if err == nil {
			content = encryptedText
			// Сохраняем IV в поле Data
			msgData, _ := json.Marshal(map[string]string{"iv": iv})
			msg := &common.Message{
				Type:      msgType,
				Sender:    c.username,
				Recipient: recipient,
				Content:   content,
				Data:      msgData,
				Token:     c.token,
			}
			common.WriteMessage(c.conn, msg)
			return
		}
	}

	msg := &common.Message{
		Type:      msgType,
		Sender:    c.username,
		Recipient: recipient,
		Content:   content,
		Token:     c.token,
	}
	common.WriteMessage(c.conn, msg)
}

func (c *ChatClient) SendTypingNotification(recipient string, isTyping bool) {
	status := "false"
	if isTyping {
		status = "true"
	}

	msg := &common.Message{
		Type:      common.MsgTyping,
		Sender:    c.username,
		Recipient: recipient,
		Content:   status,
		Token:     c.token,
	}
	common.WriteMessage(c.conn, msg)
}

func (c *ChatClient) readMessages() {
	for {
		select {
		case <-c.done:
			return
		default:
			msg, err := common.ReadMessage(c.conn)
			if err != nil {
				if c.connected {
					c.ui.ShowError("Потеряно соединение с сервером")
					c.Disconnect()
				}
				return
			}
			c.messageChan <- msg
		}
	}
}

func (c *ChatClient) processMessages() {
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.messageChan:
			c.handleMessage(msg)
		}
	}
}

func (c *ChatClient) handleMessage(msg *common.Message) {
	switch msg.Type {
	case common.MsgText:
		c.ui.ShowMessage(msg)
	case common.MsgEncrypted:
		// Пытаемся расшифровать
		if c.encryptionKey != nil && msg.Data != nil {
			var data map[string]string
			json.Unmarshal(msg.Data, &data)
			if iv, ok := data["iv"]; ok {
				decrypted, err := common.DecryptString(c.encryptionKey, msg.Content, iv)
				if err == nil {
					msg.Content = decrypted
				}
			}
		}
		c.ui.ShowMessage(msg)
	case common.MsgUserList:
		var users []string
		json.Unmarshal(msg.Data, &users)
		c.ui.UpdateUserList(users)
	case common.MsgUserJoined:
		c.ui.ShowMessage(msg)
	case common.MsgUserLeft:
		c.ui.ShowMessage(msg)
	case common.MsgError:
		c.ui.ShowError(msg.Content)
	case common.MsgPong:
		// Игнорируем pong
	default:
		log.Printf("Неизвестный тип сообщения: %s", msg.Type)
	}
}

func (c *ChatClient) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			msg := &common.Message{
				Type: common.MsgPing,
			}
			common.WriteMessage(c.conn, msg)
		}
	}
}

func (c *ChatClient) Disconnect() {
	if c.connected {
		c.connected = false
		close(c.done)

		// Отправляем сообщение о выходе
		if c.username != "" {
			msg := &common.Message{
				Type: common.MsgLogout,
			}
			common.WriteMessage(c.conn, msg)
		}

		c.conn.Close()
		log.Println("Отключено от сервера")
	}
}

func (c *ChatClient) SetEncryptionKey(key []byte) {
	c.encryptionKey = key
}

func (c *ChatClient) GetUsername() string {
	return c.username
}

func (c *ChatClient) IsConnected() bool {
	return c.connected
}

func (c *ChatClient) SetUI(ui UIHandler) {
	c.ui = ui
}
