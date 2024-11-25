package messagebroker

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

type ParagraphTask struct {
	ID         string   `json:"id"`
	Text       string   `json:"text"`
	Words      []string `json:"words"`
	SourceLang string   `json:"sourceLang"`
	TargetLang string   `json:"targetLang"`
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}

	q, err := ch.QueueDeclare(
		"paragraphs", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %v", err)
	}

	return &RabbitMQ{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (r *RabbitMQ) PublishParagraph(ctx context.Context, task ParagraphTask) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %v", err)
	}

	return r.channel.PublishWithContext(ctx,
		"",           // exchange
		r.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		})
}

func (r *RabbitMQ) ConsumeParagraphs(ctx context.Context) (<-chan ParagraphTask, error) {
	msgs, err := r.channel.Consume(
		r.queue.Name, // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %v", err)
	}

	tasks := make(chan ParagraphTask)
	go func() {
		defer close(tasks)
		for d := range msgs {
			var task ParagraphTask
			if err := json.Unmarshal(d.Body, &task); err != nil {
				continue
			}
			select {
			case tasks <- task:
				d.Ack(false)
			case <-ctx.Done():
				return
			}
		}
	}()

	return tasks, nil
}

func (r *RabbitMQ) Close() error {
	if err := r.channel.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %v", err)
	}
	if err := r.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %v", err)
	}
	return nil
}
