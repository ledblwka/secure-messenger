.PHONY: build-server build-client run-server run-client clean test fmt

# Цвета для вывода
GREEN = \033[0;32m
YELLOW = \033[1;33m
NC = \033[0m # No Color

# Пути
BIN_DIR = bin
SERVER_BIN = $(BIN_DIR)/server
CLIENT_BIN = $(BIN_DIR)/client

# Сборка сервера
build-server:
	@echo "$(YELLOW)🔨 Сборка сервера...$(NC)"
	@mkdir -p $(BIN_DIR)
	@cd cmd/server && go build -o ../../$(SERVER_BIN)
	@echo "$(GREEN)✅ Сервер собран: $(SERVER_BIN)$(NC)"

# Сборка клиента
build-client:
	@echo "$(YELLOW)🔨 Сборка клиента...$(NC)"
	@mkdir -p $(BIN_DIR)
	@cd cmd/client && go build -o ../../$(CLIENT_BIN)
	@echo "$(GREEN)✅ Клиент собран: $(CLIENT_BIN)$(NC)"

# Сборка всего
build-all: build-server build-client
	@echo "$(GREEN)🎉 Всё собрано!$(NC)"

# Быстрая сборка (без вывода)
build:
	@mkdir -p $(BIN_DIR)
	@cd cmd/server && go build -o ../../$(SERVER_BIN) > /dev/null 2>&1
	@cd cmd/client && go build -o ../../$(CLIENT_BIN) > /dev/null 2>&1

# Запуск сервера
run-server: build-server
	@echo "$(YELLOW)🚀 Запуск сервера на localhost:8080...$(NC)"
	@echo "$(YELLOW)Нажмите Ctrl+C для остановки$(NC)"
	@./$(SERVER_BIN)

# Запуск сервера на другом порту
run-server-dev:
	@./$(SERVER_BIN) -addr ":9090"

# Запуск клиента
run-client: build-client
	@echo "$(YELLOW)🚀 Запуск клиента...$(NC)"
	@./$(CLIENT_BIN)

# Запуск нескольких клиентов
run-clients:
	@echo "$(YELLOW)🚀 Запуск 3 клиентов...$(NC)"
	@for i in 1 2 3; do \
		gnome-terminal -- ./$(CLIENT_BIN) 2>/dev/null || \
		xterm -e "./$(CLIENT_BIN)" 2>/dev/null || \
		echo "Запустите клиенты вручную"; \
	done

# Очистка
clean:
	@echo "$(YELLOW)🧹 Очистка...$(NC)"
	@rm -rf $(BIN_DIR)/
	@rm -f users.json
	@rm -f *.log
	@rm -f logs/*.log 2>/dev/null || true
	@echo "$(GREEN)✅ Очищено!$(NC)"

# Форматирование кода
fmt:
	@echo "$(YELLOW)🎨 Форматирование кода...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Код отформатирован!$(NC)"

# Тестирование
test:
	@echo "$(YELLOW)🧪 Запуск тестов...$(NC)"
	@go test ./... -v

# Зависимости
deps:
	@echo "$(YELLOW)📦 Обновление зависимостей...$(NC)"
	@go mod tidy
	@go get -u golang.org/x/crypto
	@echo "$(GREEN)✅ Зависимости обновлены!$(NC)"

# Показать дерево проекта
tree:
	@echo "$(YELLOW)🌳 Структура проекта:$(NC)"
	@find . -type f -name "*.go" | sed 's|^./||' | sort

# Проверка качества кода
lint:
	@echo "$(YELLOW)🔍 Проверка стиля кода...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "Установите golangci-lint: https://golangci-lint.run/"; \
	fi

# Генерация документации
doc:
	@echo "$(YELLOW)📚 Генерация документации...$(NC)"
	@go doc -all ./...

# Показать help
help:
	@echo "$(GREEN)📋 Secure Messenger - Управление проектом$(NC)"
	@echo "==========================================="
	@echo "$(YELLOW)Основные команды:$(NC)"
	@echo "  make build         - Быстрая сборка сервера и клиента"
	@echo "  make build-all     - Сборка с подробным выводом"
	@echo "  make run-server    - Собрать и запустить сервер"
	@echo "  make run-client    - Собрать и запустить клиент"
	@echo "  make run-clients   - Запустить несколько клиентов (Linux)"
	@echo ""
	@echo "$(YELLOW)Утилиты:$(NC)"
	@echo "  make clean         - Очистить бинарники и логи"
	@echo "  make fmt           - Форматировать код"
	@echo "  make test          - Запустить тесты"
	@echo "  make deps          - Обновить зависимости"
	@echo "  make lint          - Проверить качество кода"
	@echo "  make tree          - Показать структуру проекта"
	@echo "  make help          - Показать эту справку"
	@echo ""
	@echo "$(YELLOW)Пример работы:$(NC)"
	@echo "  1. make run-server  # Запустить сервер"
	@echo "  2. make run-client  # Запустить клиент в другом терминале"
	@echo "  3. make clean       # Очистить проект"

# По умолчанию
.DEFAULT_GOAL := help