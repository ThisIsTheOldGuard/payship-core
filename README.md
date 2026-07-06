# Payship Core

Микросервисная система обработки заказов маркетплейса. Pet-проект для отработки production-практик Go, работы с распределёнными системами и инфраструктурой как код.

<a id="stack"></a>
## Стэк
- **Язык:** Go 1.24+
- **БД:** PostgreSQL 15 (Alpine)
- **Миграции:** `golang-migrate` (автоматизированы через Docker Compose)
- **Инфраструктура:** Docker, Docker Compose v2, WSL2 (Ubuntu)
- **API:** REST (планируется gRPC)
- **Очереди:** Apache Kafka (в планах)

<a id="dependencies"></a>
## 📦 Зависимости
Для локального запуска требуется:
1. **WSL2** с дистрибутивом Ubuntu (рекомендуется)
2. **Docker Desktop** + Docker Compose v2
3. **Go 1.24+** (установка: `sudo apt install golang-go`)
4. **Git**
5. **Migrate** (`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)

<a id="deploy"></a>
## Развертывание

### Клонирование и зависимости
``bash
git clone https://github.com/ThisIsTheOldGuard/payship-core.git
cd payship-core
go mod download``

### Запуск инфраструктуры
``` bash
#Запуск контейнера
docker compose up -d

# Проверка статуса контейнеров
docker compose ps

# Вход в консоль БД
docker exec -it payship-db psql -U admin -d payship_core
```

### БД & Миграции
- **Автоматическое применение:** через сервис migrate в docker-compose.yml

### 📁 Структура
```
.
├── cmd/                            # Точки входа (api)
├── internal/                       # Бизнес-логика
├── docs/                           # Полная API-документация
├── migrations/       # Миграции psql
├── docker-compose.yml
└── README.md
```

<a id="api"></a>
##  API
Полная спецификация с примерами запросов/ответов и описанием ошибок доступна в **[📖 docs/api.md](docs/api.md)**.

| Метод  | Путь                      | Описание                                                    |
|--------|---------------------------|-------------------------------------------------------------|
| `POST` | `/order`                  | [Создание заказа](docs/api.md#post-order)                   |
| `GET`  | `/order/{id}`             | [Получение по ID](docs/api.md#get-order-id)                 |
| `GET`  | `/orders`                 | [Список с пагинацией](docs/api.md#get-orders)               |
| `POST` | `order/{id}/transitions`  | [Обновление статуса](docs/api.md#post-order-id-transitions) |

<a id="api"></a>
##  Тесты
Временно тесты реализованы только для ./internal/service
```
# Запустить тесты пакета service с флагом покрытия
go test ./internal/service -cover

# HTML-отчёт
go test ./internal/service -coverprofile=coverage.out
go tool cover -html=coverage.out
xdg-open coverage.html
# Открываем результат html с возможностью просмотра покрытия кода
```

<a id="verification"></a>
## Верификация

- **REST API:** http://localhost:8080 (При работе с VPN```ip addr show eth0 | grep inet```:8080)
- **PostgreSQL** localhost:5432 (user: ```admin```, pass: ```secret```)
- **Логи:** ```docker compose logs -f```

<a id="road-map"></a>
## Road-Map
Hardening (сейчас) > API & Домен > Логирование (Прометеус) > Асинхронность & Брокер (Kafka) > CI/CD & Deploy

<a id="plans"></a>
## Будущие планы
- Прикрутить Postman
- [Использовать этот валидатор](https://github.com/go-playground/validator)