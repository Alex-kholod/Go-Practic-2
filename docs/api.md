# Список API endpoints
## Auth Service
**Base URL:** `http://localhost:8081`

### POST /v1/auth/login
Авторизация пользователя и выдача токена.

**Request (JSON):**
```json
{
  "username": "student",
  "password": "student"
}
```

**Response 200:**
```json
{
  "access_token": "demo-token",
  "token_type": "Bearer"
}
```

**Ошибки:**
- 400 — неверный формат запроса  
- 401 — неверные учетные данные  

---

### GET /v1/auth/verify
Проверка валидности токена.

**Headers:**
```
Authorization: Bearer demo-token
X-Request-ID: <uuid>
```

**Response 200:**
```json
{
  "valid": true,
  "subject": "student"
}
```

**Response 401:**
```json
{
  "valid": false,
  "error": "unauthorized"
}
```

---

## Tasks Service
**Base URL:** `http://localhost:8082`

Все запросы требуют заголовок:
```
Authorization: Bearer demo-token
```

Опционально:
```
X-Request-ID: <uuid>
```

### POST /v1/tasks
Создание задачи.

**Request:**
```json
{
  "title": "Read lecture",
  "description": "Prepare notes",
  "due_date": "2026-01-10"
}
```

**Response 201:**
```json
{
  "id": "t_001",
  "title": "Read lecture",
  "description": "Prepare notes",
  "due_date": "2026-01-10",
  "done": false
}
```

---

### GET /v1/tasks
Получение списка всех задач.

**Response 200:**
```json
[
  {"id":"t_001","title":"Read lecture","done":false},
  {"id":"t_002","title":"Do practice","done":true}
]
```

---

### GET /v1/tasks/{id}
Получение задачи по ID.

**Response 200:**
```json
{
  "id":"t_001",
  "title":"Read lecture",
  "description":"Prepare notes",
  "done":false
}
```

**Response 404:** задача не найдена

---

### PATCH /v1/tasks/{id}
Обновление задачи.

**Request:**
```json
{
  "title": "Read lecture (updated)",
  "done": true
}
```

**Response 200:** обновлённая задача  
**Response 404:** задача не найдена  

### DELETE /v1/tasks/{id}
Удаление задачи.

**Response 204:** без тела  
**Response 404:** задача не найдена  
