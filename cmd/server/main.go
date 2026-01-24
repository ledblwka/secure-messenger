package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"secure-messenger/internal/server"
)

var userManager *server.UserManager
var wsServer *server.WebSocketServer

func main() {
	// Инициализация менеджера пользователей и WebSocket сервера
	userManager = server.NewUserManager()
	wsServer = server.NewWebSocketServer(userManager)

	// Создаем демо-пользователя
	userManager.RegisterUser("demo", "demo123")
	userManager.RegisterUser("test", "test123")

	// Настройка обработки статических файлов
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Основные маршруты
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/login", serveLogin)
	http.HandleFunc("/register", serveRegister)
	http.HandleFunc("/chat", serveChat)
	http.HandleFunc("/logout", handleLogout)

	// API эндпоинты
	http.HandleFunc("/api/register", handleRegisterAPI)
	http.HandleFunc("/api/login", handleLoginAPI)
	http.HandleFunc("/api/validate", handleValidateSession)
	http.HandleFunc("/api/users", handleGetUsers)

	// WebSocket эндпоинт
	http.HandleFunc("/ws", wsServer.HandleWebSocket)

	// API для истории сообщений
	http.HandleFunc("/api/history", handleHistory)

	// Запускаем периодическую очистку сессий
	go cleanupSessions()

	// Настройка порта и хоста для Render
	port := getPort()
	host := getHost()

	log.Printf("🚀 Secure Messenger запущен на %s:%s", host, port)
	log.Printf("🌐 Откройте в браузере: http://%s:%s", getPublicHost(), port)
	log.Printf("🔗 WebSocket: ws://%s:%s/ws", getPublicHost(), port)

	// Запуск сервера
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), nil)
	if err != nil {
		log.Fatal("❌ Ошибка запуска сервера:", err)
	}
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}

func getHost() string {
	// На Render нужно слушать 0.0.0.0
	if os.Getenv("RENDER") == "true" {
		return "0.0.0.0"
	}
	return "localhost"
}

func getPublicHost() string {
	if os.Getenv("RENDER") == "true" {
		serviceName := os.Getenv("RENDER_SERVICE_NAME")
		if serviceName != "" {
			return serviceName + ".onrender.com"
		}
		return "secure-messenger.onrender.com"
	}
	return "localhost"
}

func cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		userManager.CleanupSessions()
		log.Println("🧹 Выполнена очистка просроченных сессий")
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "./web/templates/index.html")
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/templates/login.html")
}

func serveRegister(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/templates/register.html")
}

func serveChat(w http.ResponseWriter, r *http.Request) {
	// Проверка сессии через куки или заголовок
	sessionToken := getSessionToken(r)
	username, valid := userManager.ValidateSession(sessionToken)

	if !valid {
		// Редирект на страницу входа
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Обновляем время сессии
	userManager.UpdateSession(username)

	// Устанавливаем куку с токеном
	setSessionCookie(w, sessionToken)

	// Отдаем страницу чата
	http.ServeFile(w, r, "./web/templates/chat.html")
}

func handleRegisterAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Имя пользователя и пароль обязательны", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 || len(req.Username) > 20 {
		http.Error(w, "Имя пользователя должно быть от 3 до 20 символов", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "Пароль должен быть не менее 6 символов", http.StatusBadRequest)
		return
	}

	// Регистрация пользователя
	if err := userManager.RegisterUser(req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создание сессии
	sessionToken := userManager.CreateSession(req.Username)

	// Устанавливаем куку
	setSessionCookie(w, sessionToken)

	// Возвращаем успешный ответ
	response := map[string]interface{}{
		"success":      true,
		"message":      "Регистрация успешна",
		"username":     req.Username,
		"sessionToken": sessionToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Проверка учетных данных
	valid, err := userManager.ValidateCredentials(req.Username, req.Password)
	if err != nil || !valid {
		http.Error(w, "Неверное имя пользователя или пароль", http.StatusUnauthorized)
		return
	}

	// Создание сессии
	sessionToken := userManager.CreateSession(req.Username)

	// Устанавливаем куку
	setSessionCookie(w, sessionToken)

	// Возвращаем успешный ответ
	response := map[string]interface{}{
		"success":      true,
		"message":      "Вход выполнен успешно",
		"username":     req.Username,
		"sessionToken": sessionToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleValidateSession(w http.ResponseWriter, r *http.Request) {
	sessionToken := getSessionToken(r)
	username, valid := userManager.ValidateSession(sessionToken)

	response := map[string]interface{}{
		"valid":    valid,
		"username": username,
	}

	if valid {
		userManager.UpdateSession(username)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	sessionToken := getSessionToken(r)
	if _, valid := userManager.ValidateSession(sessionToken); !valid {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	users := userManager.GetAllUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	sessionToken := getSessionToken(r)
	username, valid := userManager.ValidateSession(sessionToken)
	if !valid {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	history := userManager.GetUserHistory(username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionToken := getSessionToken(r)
	if sessionToken != "" {
		userManager.Logout(sessionToken)
	}

	// Удаляем куку
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Редирект на главную страницу
	http.Redirect(w, r, "/", http.StatusFound)
}

func getSessionToken(r *http.Request) string {
	// Пробуем получить из куки
	if cookie, err := r.Cookie("session_token"); err == nil {
		return cookie.Value
	}

	// Пробуем получить из заголовка
	return r.Header.Get("X-Session-Token")
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 часа
		HttpOnly: true,
		Secure:   os.Getenv("RENDER") == "true", // Только HTTPS на продакшене
		SameSite: http.SameSiteStrictMode,
	})
}
