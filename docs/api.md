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

---

## Endpoints

<a id="post-order"></a>
### `POST /order` — Создать заказ

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
- **`201 Created`** — Заказ успешно создан
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
### `GET /order/{id}` — Получить заказ по ID

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
### `GET /orders` — Список заказов (Пагинация)

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

#### 🔹 Пример
```bash
# По умолчанию
curl "http://127.0.0.1:8080/orders"

# Пагинация
curl "http://127.0.0.1:8080/orders?page=2&limit=5"

# Ошибка валидации
curl "http://127.0.0.1:8080/orders?limit=200"
# Ответ: {"error":"limit must be between 1 and 100"}
```