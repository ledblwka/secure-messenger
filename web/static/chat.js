// frontend/js/chat.js
class SecureMessenger {
    constructor() {
        this.socket = null;
        this.currentUser = null;
        this.currentRecipient = 'all';
        this.typingTimeout = null;
        this.isTyping = false;
        
        this.init();
    }
    
    init() {
        this.checkAuth();
        this.bindEvents();
        this.loadUIState();
    }
    
    // Проверка авторизации
    checkAuth() {
        this.currentUser = localStorage.getItem('messenger_username');
        const token = localStorage.getItem('messenger_token');
        
        if (!this.currentUser || !token) {
            this.redirectToLogin();
            return;
        }
        
        this.connectWebSocket();
        this.updateUI();
        this.loadRecentMessages();
        this.loadOnlineUsers();
        
        // Периодически обновляем список пользователей
        setInterval(() => this.loadOnlineUsers(), 30000);
    }
    
    // Подключение к WebSocket
    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?token=${localStorage.getItem('messenger_token')}`;
        
        this.socket = new WebSocket(wsUrl);
        
        this.socket.onopen = () => {
            console.log('✅ WebSocket подключен');
            this.updateConnectionStatus(true);
            this.sendSystemMessage('Вы подключились к чату');
        };
        
        this.socket.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            } catch (error) {
                console.error('Ошибка парсинга сообщения:', error);
            }
        };
        
        this.socket.onclose = () => {
            console.log('❌ WebSocket отключен');
            this.updateConnectionStatus(false);
            
            // Пытаемся переподключиться через 3 секунды
            setTimeout(() => {
                if (this.currentUser) {
                    this.connectWebSocket();
                }
            }, 3000);
        };
        
        this.socket.onerror = (error) => {
            console.error('WebSocket ошибка:', error);
            this.showNotification('Ошибка соединения', 'error');
        };
    }
    
    // Обработка сообщений от WebSocket
    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'message':
                this.displayMessage(data.message);
                break;
                
            case 'user_joined':
                this.addUserToList(data.user);
                this.showSystemMessage(`${data.username} присоединился(ась) к чату`);
                break;
                
            case 'user_left':
                this.removeUserFromList(data.username);
                this.showSystemMessage(`${data.username} покинул(а) чат`);
                break;
                
            case 'user_list':
                this.updateUserList(data.users);
                break;
                
            case 'typing':
                this.showTypingIndicator(data.username, data.isTyping);
                break;
                
            case 'error':
                this.showNotification(data.message, 'error');
                break;
                
            case 'success':
                this.showNotification(data.message, 'success');
                break;
        }
    }
    
    // Отправка сообщения
    async sendMessage() {
        const messageInput = document.getElementById('messageInput');
        const content = messageInput.value.trim();
        
        if (!content) return;
        
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            const message = {
                type: 'message',
                content: content,
                recipient: this.currentRecipient,
                timestamp: new Date().toISOString()
            };
            
            this.socket.send(JSON.stringify(message));
            
            // Показываем сообщение локально сразу
            this.displayMessage({
                sender: this.currentUser,
                content: content,
                recipient: this.currentRecipient,
                timestamp: new Date().toISOString(),
                isOwn: true
            });
            
            // Очищаем поле ввода
            messageInput.value = '';
            this.updateCharCount();
            
            // Сохраняем в историю
            this.saveMessageToHistory({
                ...message,
                sender: this.currentUser,
                isOwn: true
            });
        } else {
            this.showNotification('Нет соединения с сервером', 'error');
        }
    }
    
    // Отображение сообщения в чате
    displayMessage(message) {
        const messagesContainer = document.getElementById('messagesContainer');
        const messageElement = this.createMessageElement(message);
        
        messagesContainer.appendChild(messageElement);
        this.scrollToBottom();
        
        // Сохраняем в историю
        this.saveMessageToHistory(message);
    }
    
    // Создание элемента сообщения
    createMessageElement(message) {
        const div = document.createElement('div');
        const isOwn = message.sender === this.currentUser;
        const time = new Date(message.timestamp).toLocaleTimeString([], { 
            hour: '2-digit', 
            minute: '2-digit' 
        });
        
        div.className = `message ${isOwn ? 'message-out' : 'message-in'}`;
        div.innerHTML = `
            <div class="message-header">
                <span class="message-sender">${message.sender}</span>
                <span class="message-time">${time}</span>
            </div>
            <div class="message-content">${this.escapeHtml(message.content)}</div>
            ${message.recipient !== 'all' ? 
                `<div class="message-recipient">→ ${message.recipient}</div>` : ''}
        `;
        
        return div;
    }
    
    // Системное сообщение
    showSystemMessage(text) {
        const messagesContainer = document.getElementById('messagesContainer');
        const div = document.createElement('div');
        
        div.className = 'system-message';
        div.textContent = text;
        
        messagesContainer.appendChild(div);
        this.scrollToBottom();
    }
    
    // Уведомление (toast)
    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <span>${message}</span>
            <button class="notification-close">&times;</button>
        `;
        
        document.querySelector('.notifications').appendChild(notification);
        
        notification.querySelector('.notification-close').addEventListener('click', () => {
            notification.remove();
        });
        
        // Автоматическое скрытие через 5 секунд
        setTimeout(() => notification.remove(), 5000);
    }
    
    // Индикатор набора текста
    sendTypingIndicator(isTyping) {
        if (this.isTyping === isTyping) return;
        
        this.isTyping = isTyping;
        
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({
                type: 'typing',
                isTyping: isTyping,
                recipient: this.currentRecipient
            }));
        }
    }
    
    showTypingIndicator(username, isTyping) {
        if (username === this.currentUser) return;
        
        const indicator = document.getElementById('typingIndicator');
        
        if (isTyping) {
            indicator.textContent = `${username} печатает...`;
            indicator.style.opacity = '1';
        } else {
            indicator.style.opacity = '0';
        }
    }
    
    // Обновление списка пользователей
    updateUserList(users) {
        const userList = document.getElementById('userList');
        const onlineCount = document.getElementById('onlineCount');
        
        userList.innerHTML = '';
        
        // Добавляем "Все"
        const allUser = this.createUserElement('all', 'Все пользователи', true);
        userList.appendChild(allUser);
        
        // Добавляем других пользователей
        users.forEach(user => {
            if (user.username !== this.currentUser) {
                const userElement = this.createUserElement(user.username, user.username, user.online);
                userList.appendChild(userElement);
            }
        });
        
        onlineCount.textContent = `${users.filter(u => u.online).length} онлайн`;
    }
    
    createUserElement(username, displayName, isOnline) {
        const div = document.createElement('div');
        div.className = `user-item ${this.currentRecipient === username ? 'active' : ''}`;
        div.innerHTML = `
            <div class="user-avatar">
                ${displayName.charAt(0).toUpperCase()}
            </div>
            <div class="user-info">
                <div class="username">${displayName === 'Все пользователи' ? '@all' : '@' + displayName}</div>
                <div class="user-status ${isOnline ? 'online' : 'offline'}">
                    ${isOnline ? 'в сети' : 'не в сети'}
                </div>
            </div>
        `;
        
        div.addEventListener('click', () => {
            this.selectRecipient(username, displayName);
        });
        
        return div;
    }
    
    selectRecipient(username, displayName) {
        this.currentRecipient = username;
        const recipientInput = document.getElementById('recipientInput');
        
        recipientInput.value = username === 'all' ? '@all' : `@${username}`;
        
        // Обновляем выделение
        document.querySelectorAll('.user-item').forEach(item => {
            item.classList.remove('active');
        });
        
        document.querySelectorAll('.user-item').forEach(item => {
            if (item.querySelector('.username').textContent === (username === 'all' ? '@all' : `@${username}`)) {
                item.classList.add('active');
            }
        });
        
        // Загружаем историю переписки
        this.loadConversationHistory(username);
    }
    
    // Загрузка истории сообщений
    async loadRecentMessages() {
        try {
            const response = await fetch('/api/messages/recent', {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('messenger_token')}`
                }
            });
            
            if (response.ok) {
                const messages = await response.json();
                messages.forEach(msg => this.displayMessage(msg));
            }
        } catch (error) {
            console.error('Ошибка загрузки сообщений:', error);
        }
    }
    
    async loadConversationHistory(recipient) {
        try {
            const response = await fetch(`/api/messages/conversation/${recipient}`, {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('messenger_token')}`
                }
            });
            
            if (response.ok) {
                const messages = await response.json();
                this.displayConversation(messages);
            }
        } catch (error) {
            console.error('Ошибка загрузки переписки:', error);
        }
    }
    
    displayConversation(messages) {
        const messagesContainer = document.getElementById('messagesContainer');
        messagesContainer.innerHTML = '';
        
        messages.forEach(msg => this.displayMessage(msg));
    }
    
    // Загрузка онлайн-пользователей
    async loadOnlineUsers() {
        try {
            const response = await fetch('/api/users/online', {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('messenger_token')}`
                }
            });
            
            if (response.ok) {
                const users = await response.json();
                this.updateUserList(users);
            }
        } catch (error) {
            console.error('Ошибка загрузки пользователей:', error);
        }
    }
    
    // Сохранение сообщений в localStorage
    saveMessageToHistory(message) {
        const history = JSON.parse(localStorage.getItem('message_history') || '[]');
        history.push({
            ...message,
            id: Date.now() + Math.random()
        });
        
        // Сохраняем только последние 1000 сообщений
        if (history.length > 1000) {
            history.splice(0, history.length - 1000);
        }
        
        localStorage.setItem('message_history', JSON.stringify(history));
    }
    
    // Привязка событий
    bindEvents() {
        // Отправка сообщения
        document.getElementById('sendButton').addEventListener('click', () => this.sendMessage());
        
        document.getElementById('messageInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });
        
        // Индикатор набора текста
        document.getElementById('messageInput').addEventListener('input', () => {
            this.sendTypingIndicator(true);
            
            clearTimeout(this.typingTimeout);
            this.typingTimeout = setTimeout(() => {
                this.sendTypingIndicator(false);
            }, 1000);
            
            this.updateCharCount();
        });
        
        // Выбор получателя
        document.getElementById('recipientInput').addEventListener('change', (e) => {
            const value = e.target.value.replace('@', '');
            this.selectRecipient(value, value);
        });
        
        // Поиск пользователей
        document.getElementById('searchUsers').addEventListener('input', (e) => {
            this.filterUsers(e.target.value);
        });
        
        // Выход
        document.getElementById('logoutButton').addEventListener('click', () => this.logout());
        
        // Очистка чата
        document.getElementById('clearChat').addEventListener('click', () => {
            if (confirm('Очистить историю чата?')) {
                localStorage.removeItem('message_history');
                document.getElementById('messagesContainer').innerHTML = '';
                this.showNotification('История чата очищена', 'info');
            }
        });
        
        // Переключение темы
        document.getElementById('themeToggle').addEventListener('click', () => {
            this.toggleTheme();
        });
    }
    
    filterUsers(query) {
        const userItems = document.querySelectorAll('.user-item');
        
        userItems.forEach(item => {
            const username = item.querySelector('.username').textContent.toLowerCase();
            if (username.includes(query.toLowerCase()) || query === '') {
                item.style.display = 'flex';
            } else {
                item.style.display = 'none';
            }
        });
    }
    
    // Выход из системы
    logout() {
        if (this.socket) {
            this.socket.close();
        }
        
        localStorage.removeItem('messenger_token');
        localStorage.removeItem('messenger_username');
        this.redirectToLogin();
    }
    
    redirectToLogin() {
        window.location.href = '/login.html';
    }
    
    // Обновление интерфейса
    updateUI() {
        document.getElementById('currentUsername').textContent = this.currentUser;
        document.querySelector('.user-avatar.you').textContent = this.currentUser.charAt(0).toUpperCase();
    }
    
    updateConnectionStatus(connected) {
        const status = document.getElementById('connectionStatus');
        
        if (connected) {
            status.className = 'connection-status connected';
            status.innerHTML = '<i class="fas fa-wifi"></i> В сети';
        } else {
            status.className = 'connection-status disconnected';
            status.innerHTML = '<i class="fas fa-wifi-slash"></i> Нет связи';
        }
    }
    
    updateCharCount() {
        const input = document.getElementById('messageInput');
        const counter = document.getElementById('charCounter');
        const count = input.value.length;
        
        counter.textContent = `${count}/500`;
        
        if (count > 450) {
            counter.style.color = '#ff6b6b';
        } else if (count > 400) {
            counter.style.color = '#ffd166';
        } else {
            counter.style.color = '#999';
        }
    }
    
    scrollToBottom() {
        const container = document.getElementById('messagesContainer');
        container.scrollTop = container.scrollHeight;
    }
    
    loadUIState() {
        const savedTheme = localStorage.getItem('messenger_theme') || 'light';
        document.body.setAttribute('data-theme', savedTheme);
    }
    
    toggleTheme() {
        const currentTheme = document.body.getAttribute('data-theme');
        const newTheme = currentTheme === 'light' ? 'dark' : 'light';
        
        document.body.setAttribute('data-theme', newTheme);
        localStorage.setItem('messenger_theme', newTheme);
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    // Проверяем, находимся ли мы на странице чата
    if (document.getElementById('messagesContainer')) {
        window.messenger = new SecureMessenger();
    }
    
    // Обработка формы входа
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            
            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });
                
                if (response.ok) {
                    const data = await response.json();
                    
                    localStorage.setItem('messenger_token', data.token);
                    localStorage.setItem('messenger_username', data.user.username);
                    
                    window.location.href = '/chat.html';
                } else {
                    const error = await response.json();
                    alert(error.message || 'Ошибка входа');
                }
            } catch (error) {
                alert('Ошибка подключения к серверу');
            }
        });
    }
    
    // Обработка формы регистрации
    const registerForm = document.getElementById('registerForm');
    if (registerForm) {
        registerForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const username = document.getElementById('regUsername').value;
            const password = document.getElementById('regPassword').value;
            const confirmPassword = document.getElementById('regConfirmPassword').value;
            
            if (password !== confirmPassword) {
                alert('Пароли не совпадают');
                return;
            }
            
            try {
                const response = await fetch('/api/register', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });
                
                if (response.ok) {
                    alert('Регистрация успешна! Теперь войдите в систему.');
                    window.location.href = '/login.html';
                } else {
                    const error = await response.json();
                    alert(error.message || 'Ошибка регистрации');
                }
            } catch (error) {
                alert('Ошибка подключения к серверу');
            }
        });
    }
});

// Утилиты для работы с датами
function formatTime(date) {
    return new Date(date).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function formatDate(date) {
    return new Date(date).toLocaleDateString([], { 
        day: 'numeric', 
        month: 'short' 
    });
}