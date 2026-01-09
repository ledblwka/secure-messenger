// web/main.go
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/net/websocket"
)

var store = sessions.NewCookieStore([]byte("secure-messenger-secret-key"))

type User struct {
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type Message struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type WebServer struct {
	clients   map[string]*websocket.Conn
	users     map[string]*User
	messages  []Message
	mu        sync.RWMutex
	templates *template.Template
}

func NewWebServer() *WebServer {
	// Создаем базовые шаблоны
	tmpl := template.Must(template.New("").ParseGlob("web/templates/*.html"))

	ws := &WebServer{
		clients:   make(map[string]*websocket.Conn),
		users:     make(map[string]*User),
		messages:  make([]Message, 0),
		templates: tmpl,
	}

	return ws
}

func (ws *WebServer) Run(addr string) error {
	// Статические файлы
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// WebSocket
	http.Handle("/ws", websocket.Handler(ws.handleWebSocket))

	// Маршруты
	http.HandleFunc("/", ws.handleIndex)
	http.HandleFunc("/login", ws.handleLogin)
	http.HandleFunc("/register", ws.handleRegister)
	http.HandleFunc("/logout", ws.handleLogout)
	http.HandleFunc("/api/users", ws.handleGetUsers)
	http.HandleFunc("/api/messages", ws.handleGetMessages)
	http.HandleFunc("/api/send", ws.handleSendMessage)

	log.Printf("Веб-сервер запущен на %s", addr)
	return http.ListenAndServe(addr, nil)
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)

	if username == "" {
		// Показываем главную страницу
		ws.templates.ExecuteTemplate(w, "index.html", nil)
		return
	}

	// Показываем чат
	data := map[string]interface{}{
		"Username": username,
	}
	ws.templates.ExecuteTemplate(w, "chat.html", data)
}

func (ws *WebServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ws.templates.ExecuteTemplate(w, "login.html", nil)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")

	if username == "" {
		http.Error(w, "Введите имя пользователя", http.StatusBadRequest)
		return
	}

	ws.mu.Lock()
	ws.users[username] = &User{
		Username: username,
		Online:   true,
	}
	ws.mu.Unlock()

	// Создаем сессию
	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    username,
	})
}

func (ws *WebServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ws.templates.ExecuteTemplate(w, "register.html", nil)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")

	if username == "" {
		http.Error(w, "Введите имя пользователя", http.StatusBadRequest)
		return
	}

	ws.mu.Lock()
	defer ws.mu.Unlock()

	if _, exists := ws.users[username]; exists {
		http.Error(w, "Пользователь уже существует", http.StatusBadRequest)
		return
	}

	ws.users[username] = &User{
		Username: username,
		Online:   true,
	}

	// Автоматически логиним после регистрации
	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Save(r, w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    username,
	})
}

func (ws *WebServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)

	if username != "" {
		ws.mu.Lock()
		if user, exists := ws.users[username]; exists {
			user.Online = false
		}
		ws.mu.Unlock()
	}

	session.Values["username"] = ""
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (ws *WebServer) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	users := make([]*User, 0, len(ws.users))
	for _, user := range ws.users {
		if user.Online {
			users = append(users, user)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (ws *WebServer) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws.messages)
}

func (ws *WebServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msg struct {
		Recipient string `json:"recipient"`
		Content   string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	session, _ := store.Get(r, "session")
	sender, _ := session.Values["username"].(string)

	if sender == "" {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	message := Message{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      "message",
		Sender:    sender,
		Recipient: msg.Recipient,
		Content:   msg.Content,
		Timestamp: time.Now(),
	}

	ws.mu.Lock()
	ws.messages = append(ws.messages, message)
	ws.mu.Unlock()

	// Рассылаем через WebSocket
	ws.broadcastMessage(message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      message.ID,
	})
}

func (ws *WebServer) handleWebSocket(conn *websocket.Conn) {
	defer conn.Close()

	// Получаем имя пользователя из запроса
	username := conn.Request().URL.Query().Get("user")
	if username == "" {
		return
	}

	ws.mu.Lock()
	ws.clients[username] = conn
	ws.mu.Unlock()

	log.Printf("WebSocket подключен: %s", username)

	// Читаем сообщения
	for {
		var msg map[string]interface{}
		if err := websocket.JSON.Receive(conn, &msg); err != nil {
			break
		}
	}

	ws.mu.Lock()
	delete(ws.clients, username)
	ws.mu.Unlock()

	log.Printf("WebSocket отключен: %s", username)
}

func (ws *WebServer) broadcastMessage(msg Message) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	messageData := map[string]interface{}{
		"type":      "message",
		"id":        msg.ID,
		"sender":    msg.Sender,
		"recipient": msg.Recipient,
		"content":   msg.Content,
		"time":      msg.Timestamp.Unix(),
	}

	for username, client := range ws.clients {
		// Отправляем сообщение отправителю, получателю или всем
		if msg.Recipient == "all" || msg.Sender == username || msg.Recipient == username {
			websocket.JSON.Send(client, messageData)
		}
	}
}

func main() {
	// Создаем структуру каталогов
	createDirectories()

	server := NewWebServer()

	fmt.Println("========================================")
	fmt.Println("     Secure Messenger Web Version")
	fmt.Println("     Веб-сервер запускается...")
	fmt.Println("========================================")

	// Запускаем на порту 8081
	log.Fatal(server.Run(":8081"))
}

func createDirectories() {
	// Создаем каталоги если их нет
	dirs := []string{"web/templates", "web/static"}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
	}
}
