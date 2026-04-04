package rabbit

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// публикует задачи в очередь task_jobs.
type JobProducer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	log     *zap.Logger
}

// подключается к RabbitMQ и объявляет нужные очереди.
func NewJobProducer(url string, log *zap.Logger) (*JobProducer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("amqp channel: %w", err)
	}

	if err := DeclareJobQueues(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	log.Info("job producer connected to rabbitmq",
		zap.String("component", "job_producer"),
		zap.String("queue", JobQueue),
	)

	return &JobProducer{conn: conn, channel: ch, log: log}, nil
}

// объявляет основную очередь с DLX и DLQ.
// Экспортирована — вызывается и из producer и из worker.
// Вызов идемпотентен: если очереди уже существуют — ничего не меняется.
func DeclareJobQueues(ch *amqp.Channel) error {
	// 1. DLX — dead-letter exchange типа fanout.
	if err := ch.ExchangeDeclare(
		JobDLX, "fanout",
		true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}

	// 2. DLQ — биндим к DLX.
	if _, err := ch.QueueDeclare(JobDLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := ch.QueueBind(JobDLQ, "", JobDLX, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	// 3. Основная очередь — при nack(requeue=false) сообщение уходит в DLX.
	if _, err := ch.QueueDeclare(
		JobQueue, true, false, false, false,
		amqp.Table{"x-dead-letter-exchange": JobDLX},
	); err != nil {
		return fmt.Errorf("declare job queue: %w", err)
	}

	return nil
}

// PublishJob отправляет задачу в основную очередь.
func (p *JobProducer) PublishJob(ctx context.Context, job JobMessage) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	err = p.channel.PublishWithContext(ctx,
		"", JobQueue, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    job.MessageID,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish job: %w", err)
	}

	p.log.Info("job published",
		zap.String("component", "job_producer"),
		zap.String("job", job.Job),
		zap.String("task_id", job.TaskID),
		zap.String("message_id", job.MessageID),
		zap.Int("attempt", job.Attempt),
	)
	return nil
}

// Close закрывает соединение.
func (p *JobProducer) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
