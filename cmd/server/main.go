package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"secure-messenger/internal/server"
)

var userManager *server.UserManager
var wsServer *server.WebSocketServer

func main() {
	userManager = server.NewUserManager()
	wsServer = server.NewWebSocketServer(userManager)

	// Создаем тестового пользователя для демонстрации
	userManager.RegisterUser("demo", "demo123")

	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/login", serveLogin)
	http.HandleFunc("/register", serveRegister)
	http.HandleFunc("/chat", serveChat)
	http.HandleFunc("/api/register", handleRegisterAPI)
	http.HandleFunc("/api/login", handleLoginAPI)

	http.HandleFunc("/ws", wsServer.HandleWebSocket)

	http.HandleFunc("/api/history", handleHistory)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("✅ Secure Messenger запущен на порту %s", port)
	log.Printf("📁 Откройте в браузере: http://localhost:%s", port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("❌ Ошибка сервера:", err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/templates/index.html")
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/templates/login.html")
}

func serveRegister(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/templates/register.html")
}

func serveChat(w http.ResponseWriter, r *http.Request) {
	// Проверка авторизации через сессию
	sessionToken := getSessionToken(r)
	if sessionToken == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Проверяем валидность токена
	username, valid := userManager.ValidateSession(sessionToken)
	if !valid {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Обновляем время сессии
	userManager.UpdateSession(username)

	// Передаем имя пользователя в шаблон
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, "./web/templates/chat.html")
}

func handleRegisterAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	if err := userManager.RegisterUser(req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создаем сессию
	sessionToken := userManager.CreateSession(req.Username)

	response := map[string]interface{}{
		"success":      true,
		"sessionToken": sessionToken,
		"username":     req.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	valid, err := userManager.ValidateCredentials(req.Username, req.Password)
	if err != nil || !valid {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Создаем сессию
	sessionToken := userManager.CreateSession(req.Username)

	response := map[string]interface{}{
		"success":      true,
		"sessionToken": sessionToken,
		"username":     req.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	// Проверяем сессию
	sessionToken := getSessionToken(r)
	username, valid := userManager.ValidateSession(sessionToken)
	if !valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	history := userManager.GetUserHistory(username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func getSessionToken(r *http.Request) string {
	// Пробуем получить токен из куки
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie != nil {
		return cookie.Value
	}

	// Пробуем получить из заголовка
	return r.Header.Get("X-Session-Token")
}

// Middleware для проверки сессии
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken := getSessionToken(r)
		if sessionToken == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		username, valid := userManager.ValidateSession(sessionToken)
		if !valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Добавляем имя пользователя в контекст запроса
		r.Header.Set("X-Username", username)
		next.ServeHTTP(w, r)
	}
}
