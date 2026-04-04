package rabbit

// Имена очередей для job-системы.
const (
	JobQueue = "task_jobs"     // основная очередь задач
	JobDLQ   = "task_jobs_dlq" // dead-letter queue
	JobDLX   = "task_jobs_dlx" // dead-letter exchange
)

// формат сообщения в очереди задач.
type JobMessage struct {
	Job       string `json:"job"` // тип работы: process_task
	TaskID    string `json:"task_id"`
	Attempt   int    `json:"attempt"`    // текущий номер попытки (начинается с 1)
	MessageID string `json:"message_id"` // UUID — ключ идемпотентности
	RequestID string `json:"request_id,omitempty"`
}

// максимальное число попыток до отправки в DLQ.
const MaxAttempts = 3
