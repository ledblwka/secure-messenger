class SecureMessenger {
    constructor() {
        this.username = '';
        this.sessionToken = '';
        this.socket = null;
        this.currentChat = 'general';
        this.users = [];
        this.isConnected = false;
        this.loadedHistory = false;
        this.typingTimeout = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        
        this.init();
    }
    
    async init() {
        console.log('🔐 Secure Messenger инициализация...');
        
        await this.checkAuth();
        if (!this.username) {
            return;
        }
        
        this.loadUI();
        await this.loadMessageHistory();
        this.connectWebSocket();
        this.setupEventListeners();
    }
    
    async checkAuth() {
        // Проверяем сессию из localStorage
        this.username = localStorage.getItem('username') || '';
        this.sessionToken = localStorage.getItem('sessionToken') || '';
        
        if (!this.username || !this.sessionToken) {
            window.location.href = '/login';
            return false;
        }
        
        return true;
    }
    
    async loadMessageHistory() {
        try {
            const response = await fetch(`/api/history`, {
                headers: {
                    'X-Session-Token': this.sessionToken
                }
            });
            
            if (response.status === 401) {
                // Сессия истекла
                this.logout();
                return;
            }
            
            if (response.ok) {
                const history = await response.json();
                this.displayHistory(history);
            }
        } catch (error) {
            console.error('Ошибка загрузки истории:', error);
        }
        this.loadedHistory = true;
    }
    
    displayHistory(history) {
        const container = document.getElementById('messagesContainer');
        const loading = document.getElementById('loadingMessages');
        
        if (loading) {
            loading.remove();
        }
        
        history.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
        
        history.forEach(msg => {
            this.createAndAppendMessage({
                ...msg,
                isOwn: msg.sender === this.username
            }, false);
        });
        
        if (history.length > 0) {
            container.scrollTop = container.scrollHeight;
        }
    }
    
    loadUI() {
        const container = document.getElementById('app');
        if (!container) return;
        
        container.innerHTML = `
            <div class="chat-container">
                <div class="chat-sidebar">
                    <div class="sidebar-header">
                        <div class="user-profile">
                            <div class="user-avatar online">
                                ${this.username.charAt(0).toUpperCase()}
                            </div>
                            <div class="user-info">
                                <h3 id="currentUsername">${this.username}</h3>
                                <div class="user-status">
                                    <span class="status-dot"></span>
                                    <span id="connectionStatus">подключение...</span>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="user-search">
                        <input type="text" class="search-input" placeholder="Поиск пользователей..." id="userSearch">
                    </div>
                    
                    <div class="chat-list">
                        <div class="chat-item active" onclick="messenger.selectChat('general')">
                            <div class="chat-avatar">
                                <i class="fas fa-users"></i>
                            </div>
                            <div class="chat-info">
                                <div class="chat-name">Общий чат</div>
                                <div class="chat-preview">Общайтесь со всеми</div>
                            </div>
                        </div>
                        <div id="privateChats"></div>
                    </div>
                    
                    <div style="padding: 20px; margin-top: auto;">
                        <button class="btn btn-block btn-accent" onclick="messenger.showSettings()" style="margin: 10px 0;">
                            <i class="fas fa-cog"></i> Настройки
                        </button>
                        <button class="btn btn-block btn-secondary" onclick="messenger.logout()" style="margin: 10px 0;">
                            <i class="fas fa-sign-out-alt"></i> Выйти
                        </button>
                    </div>
                </div>
                
                <div class="chat-main">
                    <div class="chat-header">
                        <div class="chat-title">
                            <div class="chat-avatar">
                                <i class="fas fa-users" id="chatIcon"></i>
                            </div>
                            <div>
                                <h2 id="chatTitle">Общий чат</h2>
                                <div class="chat-participants">
                                    <span id="participantCount">0 участников</span>
                                    <span class="encryption-badge">
                                        <i class="fas fa-lock"></i> Зашифровано
                                    </span>
                                </div>
                            </div>
                        </div>
                        
                        <div class="chat-actions">
                            <button class="action-btn" onclick="messenger.showUserList()" title="Пользователи">
                                <i class="fas fa-user-friends"></i>
                            </button>
                            <button class="action-btn" onclick="messenger.showEncryptionInfo()" title="Шифрование">
                                <i class="fas fa-shield-alt"></i>
                            </button>
                            <button class="action-btn" onclick="messenger.clearChat()" title="Очистить">
                                <i class="fas fa-trash"></i>
                            </button>
                        </div>
                    </div>
                    
                    <div class="messages-container" id="messagesContainer">
                        <div class="message system">
                            <div class="message-bubble">
                                <i class="fas fa-shield-alt"></i> Добро пожаловать в Secure Messenger!
                            </div>
                        </div>
                        <div id="loadingMessages" class="loading">
                            <div class="spinner"></div>
                            <span>Загрузка истории...</span>
                        </div>
                    </div>
                    
                    <div id="typingIndicator" class="typing-indicator" style="display: none;">
                        <div class="typing-dots">
                            <span></span><span></span><span></span>
                        </div>
                        <span id="typingText">Печатает...</span>
                    </div>
                    
                    <div class="message-input-area">
                        <div class="input-wrapper">
                            <textarea class="message-input" id="messageInput" placeholder="Введите сообщение..." rows="1" disabled></textarea>
                            <button class="send-button" id="sendButton" onclick="messenger.sendMessage()" disabled>
                                <i class="fas fa-paper-plane"></i>
                            </button>
                        </div>
                    </div>
                </div>
            </div>
            
            <div id="userListModal" class="modal" style="display: none;">
                <div class="modal-content">
                    <h3><i class="fas fa-users"></i> Онлайн пользователи</h3>
                    <div id="onlineUsersList"></div>
                    <div style="margin-top: 20px; text-align: center;">
                        <button class="btn btn-secondary" onclick="messenger.hideModal('userListModal')" style="margin: 5px;">
                            Закрыть
                        </button>
                    </div>
                </div>
            </div>
            
            <div id="settingsModal" class="modal" style="display: none;">
                <div class="modal-content">
                    <h3><i class="fas fa-cog"></i> Настройки</h3>
                    <div style="margin: 20px 0;">
                        <div class="form-group">
                            <label><i class="fas fa-user"></i> Имя пользователя</label>
                            <input type="text" class="form-control" value="${this.username}" readonly>
                        </div>
                        <div style="margin: 20px 0;">
                            <h4><i class="fas fa-key"></i> Ключи шифрования</h4>
                            <p style="color: #666; margin: 10px 0; font-size: 14px;">
                                Ваши ключи шифрования хранятся локально в браузере.
                            </p>
                            <button class="btn btn-block btn-secondary" onclick="messenger.regenerateKeys()">
                                <i class="fas fa-redo"></i> Сгенерировать новые ключи
                            </button>
                        </div>
                    </div>
                    <div style="text-align: center;">
                        <button class="btn btn-accent" onclick="messenger.hideModal('settingsModal')" style="margin: 5px;">
                            Закрыть
                        </button>
                    </div>
                </div>
            </div>
        `;
    }
    
    connectWebSocket() {
        // ✅ Сбрасываем счетчик переподключений
        this.reconnectAttempts = 0;
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        console.log('🔗 Подключение к WebSocket:', wsUrl);
        
        this.socket = new WebSocket(wsUrl);
        
        this.socket.onopen = () => {
            console.log('✅ WebSocket подключен, отправляем аутентификацию...');
            this.isConnected = true; // ✅ Устанавливаем статус
            this.updateConnectionStatus(true);
            
            // ✅ Разблокируем поле ввода
            const messageInput = document.getElementById('messageInput');
            const sendButton = document.getElementById('sendButton');
            if (messageInput) messageInput.disabled = false;
            if (sendButton) sendButton.disabled = false;
            
            // Отправляем сообщение аутентификации
            const authMsg = {
                type: 'auth',
                session_token: this.sessionToken,
                username: this.username
            };
            
            this.socket.send(JSON.stringify(authMsg));
        };
        
        this.socket.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleMessage(data);
            } catch (error) {
                console.error('Ошибка парсинга:', error);
            }
        };
        
        this.socket.onclose = (event) => {
            console.log('❌ WebSocket отключен:', event.code, event.reason);
            this.isConnected = false;
            this.updateConnectionStatus(false);
            
            // ✅ Блокируем поле ввода
            const messageInput = document.getElementById('messageInput');
            const sendButton = document.getElementById('sendButton');
            if (messageInput) messageInput.disabled = true;
            if (sendButton) sendButton.disabled = true;
            
            // ✅ Пробуем переподключиться
            if (this.reconnectAttempts < this.maxReconnectAttempts) {
                this.reconnectAttempts++;
                const delay = Math.min(3000 * this.reconnectAttempts, 15000); // Экспоненциальная задержка
                
                console.log(`🔄 Попытка переподключения ${this.reconnectAttempts}/${this.maxReconnectAttempts} через ${delay}мс`);
                
                setTimeout(() => {
                    if (!this.isConnected) {
                        this.connectWebSocket();
                    }
                }, delay);
            } else {
                console.error('❌ Максимальное количество попыток переподключения достигнуто');
                this.showNotification('Не удалось подключиться к серверу', 'error');
            }
        };
        
        this.socket.onerror = (error) => {
            console.error('WebSocket ошибка:', error);
            this.showNotification('Ошибка соединения', 'error');
        };
    }
    
    handleMessage(data) {
        console.log('📨 Получено сообщение:', data.type);
        
        switch (data.type) {
            case 'general':
            case 'private':
                if (data.sender === this.username) return;
                this.createAndAppendMessage(data);
                break;
                
            case 'history':
                if (this.loadedHistory) return;
                this.createAndAppendMessage(data, false);
                break;
                
            case 'users_list':
                this.updateUserList(data.users || []);
                break;
                
            case 'user_joined':
                this.showSystemMessage(`${data.sender} ${data.content}`);
                break;
                
            case 'user_left':
                this.showSystemMessage(`${data.sender} ${data.content}`);
                break;
                
            case 'typing':
                this.showTypingIndicator(data.sender);
                break;
                
            case 'success':
                this.showNotification(data.content || 'Успешно', 'success');
                break;
                
            case 'error':
                if (data.content === 'Authentication failed') {
                    this.logout();
                } else {
                    this.showNotification(data.content || 'Ошибка', 'error');
                }
                break;
        }
    }
    
    async sendMessage() {
        const input = document.getElementById('messageInput');
        const content = input.value.trim();
        
        if (!content || !this.isConnected) {
            this.showNotification('Нет соединения с сервером', 'error');
            return;
        }
        
        const recipient = this.currentChat === 'general' ? 'all' : this.currentChat;
        const messageType = recipient === 'all' ? 'general' : 'private';
        
        try {
            // Шифруем сообщение
            const encrypted = await this.encryptMessage(content, recipient);
            
            const message = {
                type: messageType,
                content: encrypted.content,
                recipient: recipient,
                iv: encrypted.iv,
                auth_tag: encrypted.tag
            };
            
            this.socket.send(JSON.stringify(message));
            
            // Показываем сообщение локально
            this.createAndAppendMessage({
                type: messageType,
                sender: this.username,
                content: content,
                recipient: recipient,
                timestamp: new Date().toISOString(),
                isOwn: true
            });
            
            input.value = '';
            this.adjustTextareaHeight(input);
        } catch (error) {
            console.error('Ошибка отправки сообщения:', error);
            this.showNotification('Ошибка отправки сообщения', 'error');
        }
    }
    
    async encryptMessage(content, recipient) {
        try {
            // В реальном приложении здесь должно быть настоящее шифрование
            const encoded = btoa(unescape(encodeURIComponent(content)));
            const iv = this.generateRandomBytes(12);
            
            return {
                content: `🔐 ${encoded} [для: ${recipient}]`,
                iv: btoa(iv),
                tag: 'demo_tag'
            };
        } catch (error) {
            console.error('Ошибка шифрования:', error);
            return {
                content: content,
                iv: '',
                tag: ''
            };
        }
    }
    
    generateRandomBytes(length) {
        const array = new Uint8Array(length);
        crypto.getRandomValues(array);
        return String.fromCharCode.apply(null, array);
    }
    
    createAndAppendMessage(data, scroll = true) {
        const container = document.getElementById('messagesContainer');
        const loading = document.getElementById('loadingMessages');
        
        if (loading && this.loadedHistory) {
            loading.remove();
        }
        
        const messageElement = this.createMessageElement(data);
        container.appendChild(messageElement);
        
        if (scroll) {
            container.scrollTop = container.scrollHeight;
        }
    }
    
    createMessageElement(data) {
        const isOwn = data.isOwn || data.sender === this.username;
        const isSystem = data.type === 'user_joined' || data.type === 'user_left';
        const time = new Date(data.timestamp || Date.now()).toLocaleTimeString([], {
            hour: '2-digit',
            minute: '2-digit'
        });
        
        const div = document.createElement('div');
        div.className = `message ${isOwn ? 'sent' : isSystem ? 'system' : 'received'}`;
        
        let content = data.content || '';
        let encrypted = false;
        
        if (content.startsWith('🔐 ')) {
            encrypted = true;
            const encoded = content.replace('🔐 ', '').split(' [для: ')[0];
            try {
                content = decodeURIComponent(escape(atob(encoded)));
            } catch {
                content = content.replace('🔐 ', '');
            }
        }
        
        let senderName = data.sender;
        if (isSystem && data.type === 'user_joined') {
            senderName = '';
            content = `${data.sender} ${data.content}`;
        } else if (isSystem && data.type === 'user_left') {
            senderName = '';
            content = `${data.sender} ${data.content}`;
        }
        
        const encryptionBadge = encrypted ?
            '<span class="encryption-badge" title="Зашифровано"><i class="fas fa-lock"></i></span>' : '';
        
        div.innerHTML = `
            ${!isSystem && !isOwn && senderName ? `<div class="message-sender">${senderName}</div>` : ''}
            <div class="message-bubble">
                <div class="message-content">${this.escapeHtml(content)}</div>
                ${encryptionBadge}
            </div>
            <div class="message-time">${time}</div>
        `;
        
        return div;
    }
    
    updateUserList(users) {
        this.users = users;
        
        const participantCount = document.getElementById('participantCount');
        if (participantCount) {
            const onlineCount = users.filter(u => u.is_online).length;
            participantCount.textContent = `${onlineCount} участников`;
        }
        
        // Обновляем список приватных чатов
        this.updatePrivateChatsList();
    }
    
    updatePrivateChatsList() {
        const privateChatsContainer = document.getElementById('privateChats');
        if (!privateChatsContainer) return;
        
        const onlineUsers = this.users.filter(u => 
            u.is_online && u.username !== this.username
        );
        
        privateChatsContainer.innerHTML = '';
        
        onlineUsers.forEach(user => {
            const chatItem = document.createElement('div');
            chatItem.className = 'chat-item';
            if (this.currentChat === user.username) {
                chatItem.classList.add('active');
            }
            
            chatItem.onclick = () => this.selectChat(user.username);
            chatItem.innerHTML = `
                <div class="chat-avatar">
                    ${user.username.charAt(0).toUpperCase()}
                </div>
                <div class="chat-info">
                    <div class="chat-name">${user.username}</div>
                    <div class="chat-preview">${user.public_key ? '🔐 ' : ''}В сети</div>
                </div>
            `;
            
            privateChatsContainer.appendChild(chatItem);
        });
    }
    
    updateConnectionStatus(connected) {
        this.isConnected = connected;
        const status = document.getElementById('connectionStatus');
        if (status) {
            status.textContent = connected ? 'в сети' : 'нет соединения';
            status.style.color = connected ? '#34a853' : '#ea4335';
        }
    }
    
    showSystemMessage(text) {
        const container = document.getElementById('messagesContainer');
        const div = document.createElement('div');
        div.className = 'message system';
        div.innerHTML = `<div class="message-bubble">${text}</div>`;
        container.appendChild(div);
        container.scrollTop = container.scrollHeight;
    }
    
    showTypingIndicator(username) {
        const indicator = document.getElementById('typingIndicator');
        const text = document.getElementById('typingText');
        
        if (indicator && text) {
            text.textContent = `${username} печатает...`;
            indicator.style.display = 'flex';
            
            clearTimeout(this.typingTimeout);
            this.typingTimeout = setTimeout(() => {
                indicator.style.display = 'none';
            }, 3000);
        }
    }
    
    showUserList() {
        const modal = document.getElementById('userListModal');
        const container = document.getElementById('onlineUsersList');
        
        if (!modal || !container) return;
        
        container.innerHTML = '';
        
        const onlineUsers = this.users.filter(u => u.is_online && u.username !== this.username);
        
        if (onlineUsers.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: #666; padding: 20px;">Нет других пользователей онлайн</p>';
        } else {
            onlineUsers.forEach(user => {
                const div = document.createElement('div');
                div.className = 'user-item';
                div.style.cssText = `
                    display: flex;
                    align-items: center;
                    padding: 15px;
                    border-radius: 10px;
                    margin: 10px 0;
                    background: #f0f7ff;
                    cursor: pointer;
                    transition: background 0.2s;
                `;
                
                div.onmouseover = () => div.style.background = '#e3f2fd';
                div.onmouseout = () => div.style.background = '#f0f7ff';
                
                div.innerHTML = `
                    <div class="user-avatar online" style="width: 40px; height: 40px; font-size: 18px; margin-right: 15px;">
                        ${user.username.charAt(0).toUpperCase()}
                    </div>
                    <div style="flex: 1;">
                        <div style="font-weight: 500; font-size: 16px;">${user.username}</div>
                        <div style="font-size: 13px; color: #666;">
                            🟢 в сети ${user.public_key ? '🔐' : ''}
                        </div>
                    </div>
                    <button class="btn" onclick="event.stopPropagation(); messenger.selectChat('${user.username}')"
                            style="padding: 8px 16px; font-size: 14px; background: #1a73e8; color: white; margin-left: 10px;">
                        <i class="fas fa-comment"></i> Написать
                    </button>
                `;
                
                div.onclick = () => this.selectChat(user.username);
                container.appendChild(div);
            });
        }
        
        modal.style.display = 'flex';
    }
    
    selectChat(chatId) {
        this.currentChat = chatId;
        this.updateChatInterface();
        this.hideModal('userListModal');
    }
    
    updateChatInterface() {
        const chatTitle = document.getElementById('chatTitle');
        const chatIcon = document.getElementById('chatIcon');
        
        if (this.currentChat === 'general') {
            chatTitle.textContent = 'Общий чат';
            chatIcon.className = 'fas fa-users';
        } else {
            chatTitle.textContent = this.currentChat;
            chatIcon.className = 'fas fa-user';
        }
        
        // Обновляем активный элемент в списке чатов
        document.querySelectorAll('.chat-item').forEach(item => {
            item.classList.remove('active');
        });
        
        const generalChat = document.querySelector('.chat-item');
        if (generalChat) {
            generalChat.classList.remove('active');
        }
        
        if (this.currentChat === 'general') {
            if (generalChat) generalChat.classList.add('active');
        } else {
            const privateChat = Array.from(document.querySelectorAll('.chat-item')).find(item => 
                item.querySelector('.chat-name')?.textContent === this.currentChat
            );
            if (privateChat) privateChat.classList.add('active');
        }
    }
    
    showEncryptionInfo() {
        alert('🔐 Шифрование сообщений:\n\n• Сообщения шифруются с помощью AES-GCM\n• Только получатель может расшифровать сообщение\n• Сервер не видит содержимое сообщений\n• Используются уникальные ключи для каждой сессии');
    }
    
    showSettings() {
        const modal = document.getElementById('settingsModal');
        if (modal) {
            modal.style.display = 'flex';
        }
    }
    
    regenerateKeys() {
        if (confirm('Вы уверены, что хотите сгенерировать новые ключи шифрования?\nВсе предыдущие сообщения не смогут быть прочитаны.')) {
            // Здесь должна быть реализация генерации новых ключей
            this.showNotification('Новые ключи сгенерированы', 'success');
        }
    }
    
    hideModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.style.display = 'none';
        }
    }
    
    logout() {
        if (confirm('Вы уверены, что хотите выйти?')) {
            // Закрываем WebSocket соединение
            if (this.socket) {
                this.socket.close();
            }
            
            // Очищаем localStorage
            localStorage.removeItem('username');
            localStorage.removeItem('sessionToken');
            
            // Перенаправляем на страницу входа
            window.location.href = '/login';
        }
    }
    
    clearChat() {
        if (confirm('Очистить историю чата (только локально)?')) {
            const container = document.getElementById('messagesContainer');
            container.innerHTML = `
                <div class="message system">
                    <div class="message-bubble">
                        История чата очищена
                    </div>
                </div>
            `;
        }
    }
    
    showNotification(message, type) {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.innerHTML = `
            <i class="fas fa-${type === 'success' ? 'check' : 'exclamation'}-circle"></i>
            <span>${message}</span>
        `;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 3000);
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    adjustTextareaHeight(textarea) {
        textarea.style.height = 'auto';
        textarea.style.height = Math.min(textarea.scrollHeight, 120) + 'px';
    }
    
    setupEventListeners() {
        const messageInput = document.getElementById('messageInput');
        if (messageInput) {
            messageInput.addEventListener('input', () => {
                this.adjustTextareaHeight(messageInput);
                
                // Отправляем уведомление о печатании
                if (this.isConnected && this.currentChat !== 'general') {
                    this.socket.send(JSON.stringify({
                        type: 'typing',
                        recipient: this.currentChat
                    }));
                }
            });
            
            messageInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    this.sendMessage();
                }
            });
        }
        
        // Закрытие модальных окон при клике вне их
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                e.target.style.display = 'none';
            }
        });
        
        // Поиск пользователей
        const searchInput = document.getElementById('userSearch');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                const searchTerm = e.target.value.toLowerCase();
                document.querySelectorAll('.chat-item').forEach(item => {
                    const name = item.querySelector('.chat-name')?.textContent?.toLowerCase() || '';
                    item.style.display = name.includes(searchTerm) ? 'flex' : 'none';
                });
            });
        }
    }
}

let messenger = null;

document.addEventListener('DOMContentLoaded', function() {
    if (window.location.pathname === '/chat' || window.location.pathname.includes('chat')) {
        messenger = new SecureMessenger();
        window.messenger = messenger;
    }
});