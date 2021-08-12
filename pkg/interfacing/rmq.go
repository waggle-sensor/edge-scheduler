package interfacing

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/streadway/amqp"
)

var (
	exchange = "scheduler"
)

type RabbitMQHandler struct {
	RabbitmqURI      string
	rabbitmqUsername string
	rabbitmqPassword string
	rabbitmqConn     *amqp.Connection
	rabbitmqChan     *amqp.Channel
}

func NewRabbitMQHandler(rabbitmqURI string, rabbitmqUsername string, rabbitmqPassword string) *RabbitMQHandler {
	return &RabbitMQHandler{
		RabbitmqURI:      rabbitmqURI,
		rabbitmqUsername: rabbitmqUsername,
		rabbitmqPassword: rabbitmqPassword,
	}
}

func (rh *RabbitMQHandler) Connect() error {
	amqpAddress := fmt.Sprintf("amqp://%s:%s@%s", rh.rabbitmqUsername, rh.rabbitmqPassword, rh.RabbitmqURI)
	logger.Info.Printf("Connecting to %s...", rh.RabbitmqURI)
	conn, err := amqp.Dial(amqpAddress)
	if err != nil {
		return err
	}
	rh.rabbitmqConn = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	rh.rabbitmqChan = ch
	return nil
}

func (rh *RabbitMQHandler) CreateExchange() error {
	if rh.rabbitmqConn == nil || rh.rabbitmqConn.IsClosed() {
		err := rh.Connect()
		if err != nil {
			return err
		}
	}
	err := rh.rabbitmqChan.ExchangeDeclare(
		exchange, // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	return err
}

func (rh *RabbitMQHandler) DeclareQueueAndConnectToExchange(queueName string) (*amqp.Queue, error) {
	q, err := rh.rabbitmqChan.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		true,      // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, err
	}
	err = rh.rabbitmqChan.QueueBind(
		q.Name,   // queue name
		q.Name,   // routing key
		exchange, // exchange
		false,
		nil)
	if err != nil {
		return nil, err
	}
	return &q, err
}

func (rh *RabbitMQHandler) SendYAML(routingKey string, message []byte) error {
	if rh.rabbitmqConn == nil || rh.rabbitmqConn.IsClosed() {
		err := rh.Connect()
		if err != nil {
			return err
		}
	}
	err := rh.rabbitmqChan.Publish(
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/yaml",
			Body:        message,
		},
	)
	return err
}

func (rh *RabbitMQHandler) GetReceiver(queueName string) (<-chan amqp.Delivery, error) {
	if rh.rabbitmqConn == nil || rh.rabbitmqConn.IsClosed() {
		err := rh.Connect()
		if err != nil {
			return nil, err
		}
	}

	q, err := rh.DeclareQueueAndConnectToExchange(queueName)

	msgs, err := rh.rabbitmqChan.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	return msgs, err
}
