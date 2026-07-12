# API Documentation

**Base URL:** `http://localhost:8080`  
**Content-Type:** `application/json`  
**Authentication:** None (MVP stage)

---

## Common Error Format
All error responses follow a unified JSON structure with an appropriate HTTP status code:
```json
{
  "error": "Human-readable error description"
}
```
## Common Health Format
All status responses have a single JSON structure with a corresponding HTTP status code:
```json
{
  "status": "Human-readable status description"
}
```


---

## Endpoints

<a id="post-order"></a>
### `POST /order` - Создать заказ

**Описание:** Создаёт новый заказ в системе. Присваивает статус `pending` и генерирует `id`.

#### 🔹 Request
- **Headers:** `Content-Type: application/json`
- **Body:**
```json
{
  "customer_name": "string (required, min 1 char)",
  "amount": "number (required, > 0)"
}
```

#### 🔹 Response
- **`201 Created`** - Заказ успешно создан
```json
{
  "id": 1,
  "customer_name": "Матфей",
  "amount": 1500.5,
  "status": "pending",
  "created_at": "2026-05-24T10:00:00Z"
}
```

#### 🔹 Errors
| Status | Response | Причина |
|--------|----------|---------|
| `400` | `{"error": "invalid JSON body"}` | Некорректный JSON синтаксис |
| `400` | `{"error": "customer_name is required"}` | Пустое или отсутствующее имя |
| `400` | `{"error": "amount must be greater than 0"}` | Сумма ≤ 0 |
| `405` | `Method Not Allowed` (plain text) | Использован не `POST` метод |
| `500` | `{"error": "internal error"}` | Ошибка БД / внутренняя неисправность |

#### 🔹 Пример
```bash
curl -X POST http://127.0.0.1:8080/order \
  -H "Content-Type: application/json" \
  -d '{"customer_name":"Матфей","amount":1500.50}'
```

---

<a id="get-order-id"></a>
### `GET /order/{id}` - Получить заказ по ID

**Описание:** Возвращает детали конкретного заказа по его уникальному идентификатору.

#### 🔹 Request
- **Path Parameters:** `id` (integer, required)

#### 🔹 Response
- **`200 OK`**
```json
{
  "id": 1,
  "customer_name": "МатФей",
  "amount": 1500.5,
  "status": "pending",
  "created_at": "2026-05-24T10:00:00Z"
}
```

#### 🔹 Errors
| Status | Response | Причина |
|--------|----------|---------|
| `400` | `{"error": "invalid order id"}` | `id` не является числом или пуст |
| `404` | `{"error": "order not found"}` | Заказ с таким `id` не существует |
| `405` | `Method Not Allowed` | Использован не `GET` метод |
| `500` | `{"error": "internal error"}` | Ошибка БД |

#### 🔹 Пример
```bash
curl http://127.0.0.1:8080/order/1
```

---

<a id="get-orders"></a>
### `GET /orders` - Список заказов (Пагинация)

**Описание:** Возвращает список заказов с поддержкой пагинации. Сортировка: по дате создания (`created_at DESC`).

#### 🔹 Request
- **Query Parameters:**

  | Параметр | Тип | По умолчанию | Описание |
  |----------|-----|--------------|----------|
  | `page` | integer | `1` | Номер страницы (`≥ 1`) |
  | `limit` | integer | `20` | Записей на странице (`1–100`) |

#### 🔹 Response
- **`200 OK`**
```json
{
  "items": [
    {
      "id": 2,
      "customer_name": "Тест",
      "amount": 99.99,
      "status": "pending",
      "created_at": "2026-05-24T11:00:00Z"
    },
    {
      "id": 1,
      "customer_name": "Матфей",
      "amount": 1500.5,
      "status": "pending",
      "created_at": "2026-05-24T10:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "limit": 20
}
```

#### 🔹 Errors
| Status | Response | Причина |
|--------|----------|---------|
| `400` | `{"error": "invalid page parameter"}` | `page` не является числом |
| `400` | `{"error": "invalid limit parameter"}` | `limit` не является числом |
| `400` | `{"error": "page must be >= 1"}` | Отрицательный или нулевой `page` |
| `400` | `{"error": "limit must be between 1 and 100"}` | `limit < 1` или `limit > 100` |
| `500` | `{"error": "internal error"}` | Ошибка БД |

<a id="post-order-id-transitions"></a>
### `POST /order/{id}/transitions` - Изменение статуса заказа

**Описание:** Переводит заказ в новый статус с обязательной валидацией допустимых переходов.

- **Path Parameters:** `id` (integer, required)

#### 🔹 Body
```json
{
  "name": "string (pending, processing, completed, cancelled)"
}
```

#### 🔹 Response
- **`204 Created`** - Выполнен успешный переход статуса заказа

**Ошибки:**

| Status | Response | Причина                                                    |
|--------|----------|------------------------------------------------------------|
| `400` | `{"error": "invalid order id"}` | `id` не является целым числом иоли пуст                    |
| `400` | `{"error": "invalid JSON body"}` | Некорректный JSON или отсутствует поле `name`              |
| `400` | `{"error": "invalid status value"}` | Переданное значение не входит в допустимый список статусов |
| `404` | `{"error": "order not found"}` | Заказ с таким `id` не существует                           |
| `400` | `{"error": "invalid transition from X to Y"}` | Переход запрещён                                           |
| `500` | `{"error": "internal error"}` | Ошибка БД                                                  |

**Правила переходов:**

| Текущий статус | Допустимые следующие статусы |
|----------------|------------------------------|
| `pending` | `processing`, `cancelled` |
| `processing` | `completed`, `cancelled` |
| `completed` | Переходы запрещены|
| `cancelled` | Переходы запрещены |

Возможно в будущем добавлю возможность возобновления закрытого или удаленного заказа, но скорее всего это будет реализовано через создание нового заказа.

**Пример:**
```bash
curl -X POST http://127.0.0.1:8080/order/1/transitions \
  -H "Content-Type: application/json" \
  -d '{"name":"processing"}'
# Ответ: {"order_id":1,"status":"processing","message":"transition successful"} (200)
```

## Healthpoints

<a id="get-root"></a>
### `GET /` - Главная страница

**Описание:** Возвращает главную страницу API

#### 🔹 Response
- **`200 OK`**
```json
{
  "status" : "API is running"
}
```

<a id="get-not-found"></a>
### `GET /not-found-page` - Неизвестная страница

**Описание:** Возвращает ошибку ненайденной страницы

#### 🔹 Response
- **`404 OK`**
```json
{
  "error" : "page not found"
}
```
<a id="get-health"></a>
### `GET /health` - Доступ к API

**Описание:** Возвращает доступность сервиса

#### 🔹 Response
- **`200 OK`**
```json
{
  "status" : "OK"
}
```
<a id="get-ready"></a>
### `GET /ready` - Доступ к БД.

**Описание:** Возвращает доступность БД.

#### 🔹 Response
- **`200 OK`**
```json
{
  "status" : "Ready"
}
```
#### 🔹 Errors
| Status | Response | Причина                 |
|--------|----------|-------------------------|
| `503`  | `{"error": "Database is unavailable"}` | База данных не доступна |

<a id="get-metrics"></a>
### `GET /metrics` - Метрики.

**Описание:** Возвращает метрики prometheus.

#### 🔹 Response
- **`200 OK`**
```json
{
  {...} // Метрики
}
```