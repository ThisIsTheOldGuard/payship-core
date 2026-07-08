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

# Проверить доступ к серверу приложения prometheus
docker exec -it prometheus wget -qO- http://host.docker.internal:8080/metrics | head
```
После разворачивания, авторизоваться в grafana, настроить подключение к promtheus (http://prometheus:9090), загрузить дашборды из ./grafana/dashboards/

### БД & Миграции
- **Автоматическое применение:** через сервис migrate в docker-compose.yml

### 📁 Структура
```
.
├── cmd/                            # Точки входа (api)
├── internal/                       # Бизнес-логика
├── docs/                           # Полная API-документация
├── migrations/                     # Миграции psql
├── grafana/                        # Сопутствующие файлы grafana
├── docker-compose.yml
├── prometheus.yml
└── README.md
```

<a id="api"></a>
##  API
Полная спецификация с примерами запросов/ответов и описанием ошибок доступна в **[📖 docs/api.md](docs/api.md)**.

| Метод  | Путь                     | Описание                                                    |
|--------|--------------------------|-------------------------------------------------------------|
| `POST` | `/order`                 | [Создание заказа](docs/api.md#post-order)                   |
| `GET`  | `/order/{id}`            | [Получение по ID](docs/api.md#get-order-id)                 |
| `GET`  | `/orders`                | [Список с пагинацией](docs/api.md#get-orders)               |
| `POST` | `order/{id}/transitions` | [Обновление статуса](docs/api.md#post-order-id-transitions) |
| `GET`  | `/metrics`               | end point prometheus                                        |

<a id="metrics"></a>
##  Метрики
На текущий момент реализованы метрики посредством prometheus
В будуем планируется дополнительно прикрутить Grafana, а также реализацию нагрузочного тестирования

| Имя_метрики                   | Краткое описание                                     |
|-------------------------------|------------------------------------------------------|
| http_requests_total           | Общее количество запросов к API                      |
| http_request_duration_seconds | Время выполнения запросов к API в формате интервалов |
| db_pool_active_conns          | Количество активных соединений БД                    |
| db_pool_wait_total            | Количество соединений ждущих своей очереди в БД      |
| db_query_duration_seconds            | Время обработки запросов к БД в формате интервалов   |

```curl
# Для получения конкретной метрики
curl -s http://172.18.204.27:8080/metrics | grep {Имя_метрики}

# Для тестирования db_pool_active_conns и db_pool_wait_total
for i in {1..10}; do curl -s http://localhost:8080/debug/db/slow > /dev/null & done; wait
```

<a id="tests"></a>
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
Получить локалхост при работе с wsl или VPN - localhost = ```hostname -I | awk '{print $1}'```:8080
- **REST API:** http://localhost:8080
- **PostgreSQL** localhost:5432 (user: ```admin```, pass: ```secret```)
- **Prometheus** localhost:9090/targets
- **Grafana** localhost:3000/dashboards (user: ```admin```, pass: ```admin```, после регистрации поменять пароль в .env)
- **Логи:** ```docker compose logs -f```

<a id="road-map"></a>
## Road-Map
Hardening (сейчас) > API & Домен > Логирование (Прометеус) > Асинхронность & Брокер (Kafka) > CI/CD & Deploy

<a id="plans"></a>
## Будущие планы
- Прикрутить Postman
- [Использовать этот валидатор](https://github.com/go-playground/validator)