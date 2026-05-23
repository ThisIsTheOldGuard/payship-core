# Payship Core

Микросервисная система обработки заказов маркетплейса. Pet-проект для отработки production-практик Go, работы с распределёнными системами и инфраструктурой как код.

##  Стэк
- **Язык:** Go 1.24+
- **БД:** PostgreSQL 15 (Alpine)
- **Миграции:** `golang-migrate` (автоматизированы через Docker Compose)
- **Инфраструктура:** Docker, Docker Compose v2, WSL2 (Ubuntu)
- **API:** REST (планируется gRPC)
- **Очереди:** Apache Kafka (в планах)

##  Зависимости
Для локального запуска требуется:
1. **WSL2** с дистрибутивом Ubuntu (рекомендуется)
2. **Docker Desktop** + Docker Compose v2
3. **Go 1.24+** (установка: `sudo apt install golang-go`)
4. **Git**
5. **Migrate** (`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)

## 🚀 СТАРТ

### 1. Клонирование и зависимости
``bash
git clone https://github.com/[YourUsername]/payship-core.git
cd payship-core
go mod download``

## Запуск инфраструктуры
``` bash
#Запуск контейнера
docker compose up -d

# Проверка статуса контейнеров
docker compose ps

# Вход в консоль БД
docker exec -it payship-db psql -U admin -d payship_core
```

## БД & Миграции
- **Автоматическое применение:** через сервис migrate в docker-compose.yml

## 📁 Структура
```
.
├── cmd/              # Точки входа (api, worker)
├── internal/         # Бизнес-логика
├── pkg/              # Переиспользуемые утилиты
├── configs/          # Конфигурации
├── docker-compose.yml
└── README.md
```

## Верификация

- **REST API:** http://localhost:8080
- **PostgreSQL** localhost:5432 (user: ```admin```, pass: ```secret```)
- **Логи:** ```docker compose logs -f```