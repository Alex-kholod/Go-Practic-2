# Практическое занятие №5

## HTTPS/TLS и защита от SQL-инъекций в серверном приложении

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

## Выбор варианта TLS

Выбран **Вариант 1 — TLS на NGINX** как терминаторе:

- Сервис `tasks` остаётся HTTP (`:8082`) и не знает про TLS.
- NGINX принимает HTTPS на `:8443` и проксирует трафик в `tasks`.
- Это ближе к индустриальной практике: сертификаты управляются отдельно от кода приложения.

## Генерация самоподписанного сертификата

```powershell
cd deploy/tls

# Сгенерировать ключ и сертификат через OpenSSL
& "C:\Program Files\Git\usr\bin\openssl.exe" req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 365 -subj "/CN=localhost"
```

- `key.pem` — приватный ключ
- `cert.pem` — самоподписанный сертификат

## Конфигурация NGINX (`deploy/tls/nginx.conf`)

```nginx
events {}

http {
    server {
        listen 8443 ssl;
        server_name localhost;

        ssl_certificate     /etc/nginx/tls/cert.pem;
        ssl_certificate_key /etc/nginx/tls/key.pem;

        location / {
            proxy_pass         http://tasks:8082;
            proxy_set_header   Host              $host;
            proxy_set_header   X-Forwarded-Proto https;
            proxy_set_header   X-Request-ID      $http_x_request_id;
            proxy_set_header   Authorization     $http_authorization;
        }
    }
}
```

NGINX принимает HTTPS и прокидывает заголовки `Authorization` и `X-Request-ID` в сервис.

## База данных — таблица tasks

Миграция выполняется автоматически при старте сервиса (`repository.New`):

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT        PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    done        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Демонстрация SQL-инъекции

### Уязвимый запрос (конкатенация строк)

```go
// УЯЗВИМО — пользовательский ввод вставляется напрямую в SQL:
query := "SELECT id, title FROM tasks WHERE title = '" + title + "'"
rows, _ := db.QueryContext(ctx, query)
```

При `title = ' OR '1'='1` запрос превращается в:
```sql
SELECT id, title FROM tasks WHERE title = '' OR '1'='1'
```
и возвращает **все записи** таблицы вне зависимости от фильтра.

### Безопасный вариант (параметризованный запрос)

```go
// БЕЗОПАСНО — пользовательский ввод передаётся как параметр $1:
rows, err := db.QueryContext(ctx,
    "SELECT id, title, description, done, created_at FROM tasks WHERE title ILIKE $1",
    "%"+title+"%",
)
```

Драйвер экранирует параметр автоматически — инъекция невозможна.

## Запуск

### Клонировать репозиторий

```powershell
git clone -b pz5 --single-branch https://github.com/Alex-kholod/Go-Practic-2.git
cd Go-Practic-2
go mod tidy
```

### Сгенерировать сертификат (один раз)

```powershell
cd deploy/tls
& "C:\Program Files\Git\usr\bin\openssl.exe" req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 365 -subj "/CN=localhost"
cd ../..
```

### Запустить auth отдельно

```powershell
$env:AUTH_PORT = "8081"
$env:AUTH_GRPC_PORT = "50051"
go run ./services/auth/cmd/auth
```

### tasks + PostgreSQL + NGINX через docker-compose

```powershell
cd deploy/tls
docker compose up -d
```

После запуска доступны:
- **tasks по HTTP** (внутри Docker-сети) — `tasks:8082`
- **tasks по HTTPS** (для клиентов) — https://localhost:8443

## Проверка HTTPS

### Создать задачу по HTTPS

```powershell
curl.exe -k -i -X POST https://localhost:8443/v1/tasks `
  -H "Content-Type: application/json" `
  -H "Authorization: Bearer $TOKEN" `
  -H "X-Request-ID: pz5-001" `
  -d '{"title":"SQL safe","description":"use params"}'
```

_Скриншот — успешный ответ по HTTPS:_

### Получить список задач по HTTPS

```powershell
curl.exe -k -i https://localhost:8443/v1/tasks `
  -H "Authorization: Bearer $TOKEN" `
  -H "X-Request-ID: pz5-002"
```

_Скриншот — список задач по HTTPS:_

### Поиск задачи

```powershell
curl.exe -k -s "https://localhost:8443/v1/tasks/search?title=SQL" `
  -H "Authorization: Bearer $TOKEN"
```

_Скриншот — результат поиска:_