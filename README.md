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
<img width="974" height="388" alt="image" src="https://github.com/user-attachments/assets/d26a0022-41d3-4ef2-85c3-c5523486163e" />

- **Grafana** — http://localhost:3000 (логин: `admin` / пароль: `admin`)
<img width="974" height="492" alt="image" src="https://github.com/user-attachments/assets/feb386d8-8476-4b3a-81ea-1776999352ba" />

## Проверка метрик
### Проверить endpoint /metrics напрямую

```bash
curl -s http://localhost:8082/metrics | grep http_
```

_Скриншот вывода `/metrics`:_
<img width="974" height="600" alt="image" src="https://github.com/user-attachments/assets/478490e5-4463-4629-9d3b-3f8fc59d2e34" />


### targets в Prometheus

Открыть http://localhost:9090/targets оба target имеют статус **UP**.

_Скриншот страницы Targets в Prometheus:_
<img width="974" height="496" alt="image" src="https://github.com/user-attachments/assets/ee1179a6-eecb-4cda-99e4-c3a62bd7c0cf" />

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
<img width="974" height="564" alt="image" src="https://github.com/user-attachments/assets/b1ae734f-5a1a-4555-920c-cc0f723a123c" />

## Графики в Grafana
### График 1 — RPS (запросы в секунду)

```promql
sum(rate(http_requests_total{service="tasks"}[1m])) by (route)
```

_Скриншот графика RPS:_
<img width="974" height="463" alt="image" src="https://github.com/user-attachments/assets/bd83444e-2a08-4297-9f54-5bad3daa2a5e" />


### График 2 — Ошибки (4xx и 5xx)

```promql
sum(rate(http_requests_total{service="tasks", status=~"4..|5.."}[1m])) by (status)
```

_Скриншот графика ошибок:_
<img width="974" height="497" alt="image" src="https://github.com/user-attachments/assets/19b880da-60db-4d76-9843-18b84d4badaa" />

### График 3 — Latency p95

```promql
histogram_quantile(0.95,
  sum(rate(http_request_duration_seconds_bucket{service="tasks"}[1m])) by (le, route)
)
```

_Скриншот графика latency p95:_
<img width="974" height="491" alt="image" src="https://github.com/user-attachments/assets/3f21b38a-f17d-4f2c-8e97-d081abd243da" />

