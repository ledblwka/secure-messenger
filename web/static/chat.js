// web/static/chat.js
class ChatApp {
    constructor(username) {
        this.username = username;
        this.ws = null;
        this.currentRecipient = 'all';
        this.isTyping = false;
        this.typingTimeout = null;
        this.connected = false;
        
        this.init();
    }
    
    init() {
        this.connectWebSocket();
        this.bindEvents();
        this.loadInitialData();
        
        // Периодическая проверка соединения
        setInterval(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.sendPing();
            }
        }, 30000);
    }
    
    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?user=${encodeURIComponent(this.username)}`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('✅ WebSocket подключен');
            this.connected = true;
            this.updateConnectionStatus(true);
            this.showSystemMessage('Вы подключились к чату');
        };
        
        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleMessage(data);
            } catch (error) {
                console.error('Ошибка парсинга сообщения:', error);
            }
        };
        
        this.ws.onclose = () => {
            console.log('❌ WebSocket отключен');
            this.connected = false;
            this.updateConnectionStatus(false);
            
            // Переподключение через 3 секунды
            setTimeout(() => {
                if (this.username) {
                    this.connectWebSocket();
                }
            }, 3000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket ошибка:', error);
            this.showNotification('Ошибка соединения', 'error');
        };
    }
    
    handleMessage(data) {
        switch (data.type) {
            case 'message':
                this.displayMessage(data);
                break;
                
            case 'user_joined':
                this.addUserToList(data.username);
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
                this.showTypingIndicator(data.sender, data.isTyping);
                break;
                
            case 'error':
                this.showNotification(data.message, 'error');
                break;
                
            case 'success':
                this.showNotification(data.message, 'success');
                break;
        }
    }
    
    sendMessage() {
        const messageInput = document.getElementById('messageInput');
        const content = messageInput.value.trim();
        
        if (!content) {
            this.showNotification('Введите сообщение', 'warning');
            return;
        }
        
        if (!this.connected) {
            this.showNotification('Нет соединения с сервером', 'error');
            return;
        }
        
        const message = {
            type: 'message',
            sender: this.username,
            recipient: this.currentRecipient,
            content: content,
            timestamp: new Date().toISOString()
        };
        
        this.ws.send(JSON.stringify(message));
        
        // Показываем сообщение локально
        this.displayMessage({
            ...message,
            isOwn: true
        });
        
        // Очищаем поле ввода
        messageInput.value = '';
        this.updateCharCount();
        
        // Сбрасываем индикатор набора
        this.sendTypingIndicator(false);
    }
    
    displayMessage(data) {
        const messagesContainer = document.getElementById('messagesContainer');
        if (!messagesContainer) return;
        
        const messageElement = this.createMessageElement(data);
        messagesContainer.appendChild(messageElement);
        
        // Прокрутка вниз
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
        
        // Сохраняем в историю
        this.saveToHistory(data);
    }
    
    createMessageElement(data) {
        const div = document.createElement('div');
        const isOwn = data.isOwn || data.sender === this.username;
        const time = new Date(data.timestamp).toLocaleTimeString([], { 
            hour: '2-digit', 
            minute: '2-digit' 
        });
        
        div.className = `message-bubble ${isOwn ? 'own' : 'other'}`;
        div.innerHTML = `
            <div style="font-size: 12px; opacity: 0.8; margin-bottom: 5px;">
                ${data.sender} • ${time}
            </div>
            <div style="font-size: 16px; line-height: 1.4;">
                ${this.escapeHtml(data.content)}
            </div>
            ${data.recipient !== 'all' ? 
                `<div style="font-size: 11px; opacity: 0.7; margin-top: 5px;">
                    → ${data.recipient}
                </div>` : ''}
        `;
        
        return div;
    }
    
    showSystemMessage(text) {
        const messagesContainer = document.getElementById('messagesContainer');
        if (!messagesContainer) return;
        
        const div = document.createElement('div');
        div.className = 'system-message';
        div.style.cssText = `
            text-align: center;
            color: #666;
            font-size: 14px;
            margin: 10px 0;
            padding: 5px;
            font-style: italic;
        `;
        div.textContent = text;
        
        messagesContainer.appendChild(div);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
    
    showNotification(message, type = 'info') {
        console.log(`${type.toUpperCase()}: ${message}`);
        // Можно добавить toast уведомления позже
    }
    
    updateUserList(users) {
        const userList = document.getElementById('userList');
        if (!userList) return;
        
        userList.innerHTML = '';
        
        // Добавляем "Все пользователи"
        const allUser = this.createUserElement('all', 'Все пользователи', true);
        userList.appendChild(allUser);
        
        // Добавляем остальных пользователей
        users.forEach(user => {
            if (user.username !== this.username) {
                const userElement = this.createUserElement(
                    user.username, 
                    user.username, 
                    user.online
                );
                userList.appendChild(userElement);
            }
        });
        
        // Обновляем счетчик онлайн
        const onlineCount = document.getElementById('onlineCount');
        if (onlineCount) {
            const onlineUsers = users.filter(u => u.online).length;
            onlineCount.textContent = `${onlineUsers} онлайн`;
        }
    }
    
    createUserElement(username, displayName, isOnline) {
        const div = document.createElement('div');
        div.className = 'user-item';
        div.style.cssText = `
            display: flex;
            align-items: center;
            padding: 12px 15px;
            border-radius: 12px;
            margin-bottom: 8px;
            cursor: pointer;
            transition: all 0.2s;
            background: ${this.currentRecipient === username ? 'rgba(255, 255, 255, 0.2)' : 'transparent'};
        `;
        
        div.innerHTML = `
            <div style="
                width: 40px;
                height: 40px;
                border-radius: 50%;
                background: ${isOnline ? 'linear-gradient(135deg, #4CAF50, #8BC34A)' : 'linear-gradient(135deg, #9E9E9E, #757575)'};
                display: flex;
                align-items: center;
                justify-content: center;
                color: white;
                font-weight: bold;
                margin-right: 15px;
            ">
                ${displayName.charAt(0).toUpperCase()}
            </div>
            <div style="flex: 1;">
                <div style="font-weight: 600; color: white;">
                    ${displayName === 'Все пользователи' ? '@all' : '@' + displayName}
                </div>
                <div style="font-size: 12px; color: ${isOnline ? '#C8E6C9' : '#E0E0E0'};">
                    ${isOnline ? 'в сети' : 'не в сети'}
                </div>
            </div>
            ${isOnline ? 
                `<div style="
                    width: 10px;
                    height: 10px;
                    border-radius: 50%;
                    background: #4CAF50;
                "></div>` : ''}
        `;
        
        div.addEventListener('click', () => {
            this.selectRecipient(username, displayName);
        });
        
        return div;
    }
    
    selectRecipient(username, displayName) {
        this.currentRecipient = username;
        
        // Обновляем UI
        const recipientInput = document.getElementById('recipientInput');
        if (recipientInput) {
            recipientInput.value = username === 'all' ? '@all' : `@${username}`;
        }
        
        const currentRecipientSpan = document.getElementById('currentRecipient');
        if (currentRecipientSpan) {
            currentRecipientSpan.textContent = username === 'all' ? '@all (всем)' : `@${username}`;
        }
        
        // Перезагружаем историю переписки
        this.loadConversationHistory(username);
        
        // Обновляем выделение в списке пользователей
        this.updateUserSelection();
    }
    
    updateUserSelection() {
        // Реализуйте логику выделения активного пользователя
    }
    
    sendTypingIndicator(isTyping) {
        if (this.isTyping === isTyping) return;
        
        this.isTyping = isTyping;
        
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'typing',
                sender: this.username,
                recipient: this.currentRecipient,
                isTyping: isTyping
            }));
        }
    }
    
    showTypingIndicator(sender, isTyping) {
        const indicator = document.getElementById('typingIndicator');
        if (!indicator || sender === this.username) return;
        
        if (isTyping && sender) {
            indicator.textContent = `${sender} печатает...`;
            indicator.style.opacity = '1';
        } else {
            indicator.style.opacity = '0';
        }
    }
    
    sendPing() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: 'ping' }));
        }
    }
    
    updateConnectionStatus(connected) {
        const statusElement = document.getElementById('connectionStatus');
        if (!statusElement) return;
        
        if (connected) {
            statusElement.innerHTML = '<i class="fas fa-circle" style="color: #4CAF50; margin-right: 5px;"></i> В сети';
            statusElement.style.color = '#4CAF50';
        } else {
            statusElement.innerHTML = '<i class="fas fa-circle" style="color: #F44336; margin-right: 5px;"></i> Нет связи';
            statusElement.style.color = '#F44336';
        }
    }
    
    updateCharCount() {
        const input = document.getElementById('messageInput');
        const counter = document.getElementById('charCount');
        if (!input || !counter) return;
        
        const count = input.value.length;
        counter.textContent = count;
        
        if (count > 450) {
            counter.style.color = '#F44336';
        } else if (count > 400) {
            counter.style.color = '#FF9800';
        } else {
            counter.style.color = '#666';
        }
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    saveToHistory(message) {
        // Сохраняем в localStorage для истории
        const history = JSON.parse(localStorage.getItem('chat_history') || '[]');
        history.push({
            ...message,
            id: Date.now()
        });
        
        // Храним только последние 500 сообщений
        if (history.length > 500) {
            history.splice(0, history.length - 500);
        }
        
        localStorage.setItem('chat_history', JSON.stringify(history));
    }
    
    loadInitialData() {
        this.loadRecentMessages();
        this.loadOnlineUsers();
    }
    
    async loadRecentMessages() {
        try {
            // Загружаем историю из localStorage
            const history = JSON.parse(localStorage.getItem('chat_history') || '[]');
            
            // Показываем последние 50 сообщений
            const recentMessages = history.slice(-50);
            recentMessages.forEach(msg => this.displayMessage(msg));
            
        } catch (error) {
            console.error('Ошибка загрузки сообщений:', error);
        }
    }
    
    async loadConversationHistory(recipient) {
        const messagesContainer = document.getElementById('messagesContainer');
        if (!messagesContainer) return;
        
        // Очищаем только если переключились на другого пользователя
        messagesContainer.innerHTML = '';
        
        // Загружаем историю из localStorage
        const history = JSON.parse(localStorage.getItem('chat_history') || '[]');
        
        // Фильтруем сообщения для этого получателя
        const conversation = history.filter(msg => 
            (msg.sender === this.username && msg.recipient === recipient) ||
            (msg.sender === recipient && msg.recipient === this.username) ||
            (recipient === 'all' && msg.recipient === 'all')
        );
        
        // Показываем сообщения
        conversation.forEach(msg => this.displayMessage(msg));
        
        if (conversation.length === 0) {
            this.showSystemMessage(recipient === 'all' ? 
                'Нет сообщений в общем чате. Будьте первым!' : 
                `Начните диалог с @${recipient}`
            );
        }
    }
    
    async loadOnlineUsers() {
        try {
            const response = await fetch('/api/users');
            if (response.ok) {
                const users = await response.json();
                this.updateUserList(users);
            }
        } catch (error) {
            console.error('Ошибка загрузки пользователей:', error);
        }
    }
    
    bindEvents() {
        // Отправка сообщения
        const sendButton = document.getElementById('sendButton');
        const messageInput = document.getElementById('messageInput');
        
        if (sendButton && messageInput) {
            sendButton.addEventListener('click', () => this.sendMessage());
            
            messageInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    this.sendMessage();
                }
            });
            
            messageInput.addEventListener('input', () => {
                this.sendTypingIndicator(true);
                
                clearTimeout(this.typingTimeout);
                this.typingTimeout = setTimeout(() => {
                    this.sendTypingIndicator(false);
                }, 1000);
                
                this.updateCharCount();
            });
            
            messageInput.addEventListener('blur', () => {
                this.sendTypingIndicator(false);
            });
        }
        
        // Выход
        const logoutButton = document.getElementById('logoutButton');
        if (logoutButton) {
            logoutButton.addEventListener('click', () => this.logout());
        }
        
        // Очистка чата
        const clearChatButton = document.getElementById('clearChat');
        if (clearChatButton) {
            clearChatButton.addEventListener('click', () => {
                if (confirm('Очистить историю чата?')) {
                    localStorage.removeItem('chat_history');
                    const messagesContainer = document.getElementById('messagesContainer');
                    if (messagesContainer) {
                        messagesContainer.innerHTML = '';
                        this.showSystemMessage('История чата очищена');
                    }
                }
            });
        }
        
        // Обновление списка пользователей
        const refreshUsersButton = document.getElementById('refreshUsers');
        if (refreshUsersButton) {
            refreshUsersButton.addEventListener('click', () => this.loadOnlineUsers());
        }
    }
    
    logout() {
        if (confirm('Выйти из аккаунта?')) {
            window.location.href = '/logout';
        }
    }
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    // Проверяем, есть ли пользователь
    const currentUser = document.querySelector('.username')?.textContent || 
                       localStorage.getItem('currentUser');
    
    if (currentUser && window.location.pathname.includes('chat')) {
        window.chatApp = new ChatApp(currentUser);
    }
});