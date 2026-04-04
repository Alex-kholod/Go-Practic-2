
# Практическое занятие №16

## Публикация приложения в Kubernetes

**Студент:** Холодков А.Д.  
**Группа:** ЭФМО-01-25

## Kubernetes стенд

Используется **minikube** — локальный однонодовый кластер.

```bash
# Установка minikube (Windows)
winget install Kubernetes.minikube

# Запуск кластера
minikube start --driver=docker

# Проверка
kubectl cluster-info
kubectl get nodes
```

_Скриншот — `kubectl get nodes`, нода в статусе Ready:_

![kubectl get nodes](docs/images/nodes.png)

## образ в кластер

При использовании minikube образ загружается напрямую из локального Docker — без внешнего registry:

```bash
# Сборка образа (из корня репозитория)
docker build -t techip-tasks:0.1 -f services/tasks/Dockerfile .

# Загрузка образа в minikube
minikube image load techip-tasks:0.1

# Проверка что образ доступен
minikube image ls | grep techip-tasks
```

В `deployment.yaml` указано `imagePullPolicy: Never` — Kubernetes берёт образ только из локального кэша, не пытается скачать из registry.

## Структура манифестов

```
deploy/k8s/
├── configmap.yaml    ← несекретная конфигурация
├── deployment.yaml   ← описание подов и контейнеров
└── service.yaml      ← доступ к подам внутри кластера
```

## ConfigMap — конфигурация приложения

`configmap.yaml` содержит несекретные параметры окружения:

| Ключ | Значение |
|------|----------|
| `TASKS_PORT` | `8082` |
| `INSTANCE_ID` | `tasks-k8s` |
| `AUTH_GRPC_ADDR` | `auth:50051` |
| `DATABASE_URL` | `postgres://...@postgres:5432/tasks` |
| `REDIS_ADDRS` | `redis:6379` |
| `RABBIT_URL` | `amqp://guest:guest@rabbitmq:5672/` |

**Отличие ConfigMap от Secret:** ConfigMap хранит обычный текст, Secret — base64-закодированные данные. Пароли к БД в production должны быть в Secret, не в ConfigMap.

## Deployment

Ключевые параметры `deployment.yaml`:

| Параметр | Значение | Назначение |
|----------|----------|------------|
| `replicas` | `2` | Две реплики для отказоустойчивости |
| `image` | `techip-tasks:0.1` | Фиксированный тег (не latest) |
| `imagePullPolicy` | `Never` | Берём из локального кэша minikube |
| `envFrom.configMapRef` | `tasks-config` | Все переменные из ConfigMap |
| `strategy.type` | `RollingUpdate` | Zero-downtime обновление |

### Readiness probe

```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8082
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 3
```

Kubernetes направляет трафик на под только когда `/health` возвращает 200. При проблемах под убирается из балансировки.

### Liveness probe

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8082
  initialDelaySeconds: 15
  periodSeconds: 20
  failureThreshold: 3
```

При трёх неудачных проверках подряд Kubernetes перезапускает контейнер. `initialDelaySeconds: 15` — даём время на подключение к БД и Redis.

**Разница:** readiness говорит «готов ли под принимать трафик», liveness — «жив ли процесс». Оба используют `/health`.

## Применение манифестов

```bash
# Применяем в правильном порядке: сначала ConfigMap
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml

# Или все сразу
kubectl apply -f deploy/k8s/
```

## Проверка состояния

```bash
# Список подов
kubectl get pods

# Детальная информация о поде (включая события и probe статус)
kubectl describe pod <pod-name>

# Список сервисов
kubectl get svc

# Логи контейнера
kubectl logs <pod-name>
kubectl logs -f <pod-name>   # follow
```

_Скриншот — `kubectl get pods`, оба пода в статусе Running:_

![kubectl get pods](docs/images/pods.png)

_Скриншот — `kubectl describe pod`, секция Conditions и Events:_

![kubectl describe pod](docs/images/describe.png)

## Масштабирование

```bash
# Увеличить число реплик до 3
kubectl scale deployment tasks --replicas=3

# Проверить
kubectl get pods

# Уменьшить обратно до 2
kubectl scale deployment tasks --replicas=2
```

_Скриншот — `kubectl get pods` с 3 репликами:_

![scale 3 replicas](docs/images/scale.png)

## Обновление образа

```bash
# Собрать новый образ с новым тегом
docker build -t techip-tasks:0.2 -f services/tasks/Dockerfile .
minikube image load techip-tasks:0.2

# Обновить тег в deployment.yaml (image: techip-tasks:0.2)
kubectl apply -f deploy/k8s/deployment.yaml

# Kubernetes выполнит rolling update — поды заменяются постепенно
kubectl rollout status deployment/tasks

# Откат если что-то пошло не так
kubectl rollout undo deployment/tasks
```