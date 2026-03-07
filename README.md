# Практическое занятие №4

## Тема: Настройка Prometheus + Grafana для метрик. Интеграция с приложением

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

## Описание добавленных метрик

В сервисы `auth` и `tasks` добавлены три группы метрик через middleware `shared/middleware/metrics.go`:

| Метрика | Тип | Labels | Описание |
|---------|-----|--------|----------|
| `http_requests_total` | Counter | `service`, `method`, `route`, `status` | Общее количество завершённых запросов |
| `http_request_duration_seconds` | Histogram | `service`, `method`, `route` | Длительность обработки запроса. Bucket'ы: 0.01, 0.05, 0.1, 0.3, 1, 3 с |
| `http_in_flight_requests` | Gauge | `service` | Текущее число запросов в обработке |

**Нормализация route:** путь `/v1/tasks/abc123` заменяется на `/v1/tasks/:id`, чтобы не раздувать кардинальность меток.

Метрики доступны на endpoint `/metrics` каждого сервиса без авторизации.

Порядок middleware в обоих сервисах:
```
RequestID → Metrics → AccessLog → handlers
```

## Установка и запуск
```bash
git clone -b pz4 --single-branch https://github.com/Alex-kholod/Go-Practic-2.git
cd Go-Practic-2
go mod tidy
```

### Запустить сервисы

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

### Запустить мониторинг

```bash
cd deploy/monitoring
docker compose up -d
```

После запуска доступны:
- **Prometheus** — http://localhost:9090
- **Grafana** — http://localhost:3000 (логин: `admin` / пароль: `admin`)

## Проверка метрик
### Проверить endpoint /metrics напрямую

```bash
curl -s http://localhost:8082/metrics | grep http_
```

_Скриншот вывода `/metrics`:_

### targets в Prometheus

Открыть http://localhost:9090/targets оба target имеют статус **UP**.

_Скриншот страницы Targets в Prometheus:_

### Генерация нагрузки для демонстрации метрик

Получить токен:
```bash
$response = curl.exe -s -X POST http://localhost:8081/v1/auth/login `
  -H "Content-Type: application/json" `
  -d '{"username":"student","password":"student"}'

$TOKEN = ($response | ConvertFrom-Json).token
Write-Host "Token: $TOKEN"
```

Серия успешных запросов (50 штук):
```bash
for ($i = 1; $i -le 50; $i++) {
  curl.exe -s http://localhost:8082/v1/tasks `
    -H "Authorization: Bearer $TOKEN" | Out-Null
}
```

Серия запросов с неверным токеном (20 штук — для генерации 401):
```bash
for ($i = 1; $i -le 20; $i++) {
  curl.exe -s http://localhost:8082/v1/tasks `
    -H "Authorization: Bearer wrong-token" | Out-Null
}
```
## Графики в Grafana
### График 1 — RPS (запросы в секунду)

```promql
sum(rate(http_requests_total{service="tasks"}[1m])) by (route)
```

_Скриншот графика RPS:_

### График 2 — Ошибки (4xx и 5xx)

```promql
sum(rate(http_requests_total{service="tasks", status=~"4..|5.."}[1m])) by (status)
```

_Скриншот графика ошибок:_

### График 3 — Latency p95

```promql
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{service="tasks"}[1m])) by (le, route)
)
```

_Скриншот графика latency p95:_

