package interfacing

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/streadway/amqp"
)

type RabbitMQHandler struct {
	RabbitmqURI      string
	rabbitmqUsername string
	rabbitmqPassword string
	rabbitmqConn     *amqp.Connection
	rabbitmqChan     *amqp.Channel
	appID            string
}

func NewRabbitMQHandler(rabbitmqURI string, rabbitmqUsername string, rabbitmqPassword string, appID string) *RabbitMQHandler {
	return &RabbitMQHandler{
		RabbitmqURI:      rabbitmqURI,
		rabbitmqUsername: rabbitmqUsername,
		rabbitmqPassword: rabbitmqPassword,
		appID:            appID,
	}
}

func (rh *RabbitMQHandler) Connect() error {
	amqpAddress := fmt.Sprintf("amqp://%s:%s@%s", rh.rabbitmqUsername, rh.rabbitmqPassword, rh.RabbitmqURI)
	logger.Debug.Printf("Connecting to %s...", rh.RabbitmqURI)
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

func (rh *RabbitMQHandler) CreateExchange(exchange string) error {
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

func (rh *RabbitMQHandler) DeclareQueueAndConnectToExchange(exchangeName string, queueName string) (*amqp.Queue, error) {
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
		q.Name,       // queue name
		q.Name,       // routing key
		exchangeName, // exchange
		false,
		nil)
	if err != nil {
		return nil, err
	}
	return &q, err
}

func (rh *RabbitMQHandler) SendYAML(routingKey string, message []byte) error {
	exchange := "scheduler"
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

// SendWaggleMessage delivers a Waggle message to Waggle data pipeline
//
// The message is sent to the "to-validator" exchange
func (rh *RabbitMQHandler) SendWaggleMessage(message *datatype.WaggleMessage, scope string) error {
	logger.Debug.Println(string(datatype.Dump(message)))
	exchange := "to-validator"
	if rh.rabbitmqConn == nil || rh.rabbitmqConn.IsClosed() {
		err := rh.Connect()
		if err != nil {
			return err
		}
	}
	err := rh.rabbitmqChan.Publish(
		exchange,
		scope,
		false,
		false,
		amqp.Publishing{
			Body:         datatype.Dump(message),
			DeliveryMode: 2,
			UserId:       rh.rabbitmqUsername,
			AppId:        rh.appID,
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
	msgs, err := rh.rabbitmqChan.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	return msgs, err
}
