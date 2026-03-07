# Практическое занятие №3

## Тема: Логирование с помощью zap. Ведение структурированных логов

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

---

## Выбор логгера

Выбран **zap** (go.uber.org/zap):

- Нативный JSON-формат «из коробки» — не нужна дополнительная настройка форматтера.  
- Работает быстрее logrus за счёт zero-allocation API.  
- Широко применяется в production-системах, хорошо документирован.

---

## Стандарт полей логов

Каждая запись включает следующие поля:

| Поле | Тип | Описание |
|------|-----|----------|
| `ts` | string | ISO-8601 время события (выставляется zap автоматически) |
| `level` | string | Уровень: debug / info / warn / error |
| `service` | string | Имя сервиса: `auth` или `tasks` |
| `msg` | string | Краткое описание события |
| `request_id` | string | Идентификатор запроса (X-Request-ID) |
| `method` | string | HTTP метод (GET, POST, …) |
| `path` | string | Путь запроса (/v1/tasks) |
| `status` | int | Код HTTP-ответа |
| `duration_ms` | int64 | Время обработки в миллисекундах |
| `error` | string | Текст ошибки (только при уровне warn/error) |
| `component` | string | Слой, в котором произошло событие (handler, auth_client, …) |

**Запрещено логировать:** пароли, токены, секреты, cookies. Если нужно отметить наличие токена — логируется `has_auth=true`.

## Установка и запуск
```bash
git clone -b pz3 --single-branch https://github.com/Alex-kholod/Go-Practic-2.git
cd Go-Practic-2
go mod tidy
```

**Сервис 1 — auth:**
```bash
export AUTH_PORT=8081
export AUTH_GRPC_PORT=50051
go run ./services/auth/cmd/auth
```

**Сервис 2 — tasks:**
```bash
export TASKS_PORT=8082
export AUTH_GRPC_ADDR=localhost:50051
go run ./services/tasks/cmd/tasks
```

## Проверка работы
### Получить токен

```bash
curl -s -X POST http://localhost:8081/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"student","password":"student"}'
```

_Скриншот:_

### Создать задачу с X-Request-ID

```bash
curl -i -X POST http://localhost:8082/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: pz3-001" \
  -d '{"title":"Logs","description":"Implement zap"}'
```

_Скриншот ответа:_

_Скриншот логов tasks-сервиса:_

_Скриншот логов auth-сервиса:_

### Запрос с неверным токеном

```bash
curl -i http://localhost:8082/v1/tasks \
  -H "Authorization: Bearer bad-token" \
  -H "X-Request-ID: pz3-err"
```

_Скриншот ответа и логов с уровнем warn:_

## Примеры лог-событий

### Успешный запрос

```json
{"level":"info","ts":"2026-03-07T14:28:54.763+0300","caller":"http/handlers.go:77","msg":"tasks: task created","service":"tasks","request_id":"8cd66e28dd9125ba","component":"handler","username":"student","task_id":"t_001","title":"Logs"}
```

### Запрос с ошибкой

```json
{"level":"warn","ts":"2026-03-07T14:27:14.343+0300","caller":"http/handlers.go:34","msg":"tasks: unauthorized","service":"tasks","request_id":"dc01f994662b8067","component":"handler","method":"POST","path":"/v1/tasks"}
```