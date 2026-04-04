package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	// QueueName — имя очереди для событий задач.
	QueueName = "task_events"
)

// TaskEvent — формат сообщения публикуемого в очередь.
type TaskEvent struct {
	Event     string `json:"event"` // тип события: task.created / task.updated / task.deleted
	TaskID    string `json:"task_id"`
	Timestamp string `json:"ts"`
	RequestID string `json:"request_id,omitempty"`
	Producer  string `json:"producer"` // имя сервиса-источника
}

// Producer публикует события в RabbitMQ.
type Producer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	log     *zap.Logger
}

// NewProducer подключается к RabbitMQ и объявляет очередь.
func NewProducer(url string, log *zap.Logger) (*Producer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("amqp channel: %w", err)
	}

	// Объявляем durable очередь — она переживает рестарт брокера.
	_, err = ch.QueueDeclare(
		QueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("queue declare: %w", err)
	}

	log.Info("connected to rabbitmq",
		zap.String("component", "producer"),
		zap.String("queue", QueueName),
	)

	return &Producer{conn: conn, channel: ch, log: log}, nil
}

// Publish отправляет событие в очередь task_events.
func (p *Producer) Publish(ctx context.Context, event TaskEvent) {
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(event)
	if err != nil {
		p.log.Error("rabbit: marshal event failed",
			zap.String("component", "producer"),
			zap.String("error", err.Error()),
		)
		return
	}

	err = p.channel.PublishWithContext(ctx,
		"",        // exchange — пустой, публикуем напрямую в очередь
		QueueName, // routing key = имя очереди
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // сообщение переживает рестарт брокера
			Body:         body,
		},
	)
	if err != nil {
		p.log.Error("rabbit: publish failed",
			zap.String("component", "producer"),
			zap.String("event", event.Event),
			zap.String("task_id", event.TaskID),
			zap.String("error", err.Error()),
		)
		return
	}

	p.log.Info("rabbit: event published",
		zap.String("component", "producer"),
		zap.String("event", event.Event),
		zap.String("task_id", event.TaskID),
		zap.String("request_id", event.RequestID),
	)
}

// Close закрывает соединение с RabbitMQ.
func (p *Producer) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
