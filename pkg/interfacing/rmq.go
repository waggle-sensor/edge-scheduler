package interfacing

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/streadway/amqp"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"gopkg.in/cenkalti/backoff.v1"
)

type RabbitMQMessageWrapper struct {
	DestName    string
	RoutingKey  string
	Body        []byte
	ContentType string
}

func NewRabbitMQMessageWrapper(to string, scope string, body []byte, contentType string) *RabbitMQMessageWrapper {
	return &RabbitMQMessageWrapper{
		DestName:    to,
		RoutingKey:  scope,
		Body:        body,
		ContentType: contentType,
	}
}

type RabbitMQHandler struct {
	RabbitmqURI      string
	rabbitmqUsername string
	rabbitmqPassword string
	cacertPath       string
	rabbitmqConn     *amqp.Connection
	rabbitmqChan     *amqp.Channel
	appID            string
	chanToPublish    chan RabbitMQMessageWrapper
}

func NewRabbitMQHandler(rabbitmqURI string, rabbitmqUsername string, rabbitmqPassword string, cacertPath string, appID string) *RabbitMQHandler {
	return &RabbitMQHandler{
		RabbitmqURI:      rabbitmqURI,
		rabbitmqUsername: rabbitmqUsername,
		rabbitmqPassword: rabbitmqPassword,
		cacertPath:       cacertPath,
		appID:            appID,
		chanToPublish:    make(chan RabbitMQMessageWrapper, 100),
	}
}

func (rh *RabbitMQHandler) Connect() error {
	// If cacert is given it attempts TLS connection
	if rh.cacertPath != "" {
		caCert, err := ioutil.ReadFile(rh.cacertPath)
		if err != nil {
			return err
		}
		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(caCert)
		tlsConf := &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: true,
		}
		amqpAddress := fmt.Sprintf("amqps://%s:%s@%s", rh.rabbitmqUsername, rh.rabbitmqPassword, rh.RabbitmqURI)
		logger.Debug.Printf("Connecting to %s...", rh.RabbitmqURI)
		conn, err := amqp.DialTLS(amqpAddress, tlsConf)
		if err != nil {
			return err
		}
		rh.rabbitmqConn = conn
		ch, err := conn.Channel()
		if err != nil {
			return err
		}
		rh.rabbitmqChan = ch
	} else {
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
	}
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

func (rh *RabbitMQHandler) DeclareQueueAndConnectToExchange(exchangeName string, queueName string, topicToSubscribe string) (*amqp.Queue, error) {
	if rh.rabbitmqConn == nil || rh.rabbitmqConn.IsClosed() {
		err := rh.Connect()
		if err != nil {
			return nil, err
		}
	}
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
		q.Name,           // queue name
		topicToSubscribe, // routing key
		exchangeName,     // exchange
		false,
		nil)
	if err != nil {
		return nil, err
	}
	return &q, err
}

func (rh *RabbitMQHandler) publish(m RabbitMQMessageWrapper) error {
	if rh.rabbitmqConn == nil || rh.rabbitmqConn.IsClosed() {
		err := rh.Connect()
		if err != nil {
			return err
		}
	}
	err := rh.rabbitmqChan.Publish(
		m.DestName,
		m.RoutingKey,
		false,
		false,
		amqp.Publishing{
			Body:         m.Body,
			DeliveryMode: 2,
			UserId:       rh.rabbitmqUsername,
			AppId:        rh.appID,
			ContentType:  m.ContentType,
		},
	)
	return err
}

// SendWaggleMessageOnNode delivers a Waggle message to Waggle data pipeline inside a node
//
// The message is sent to the "to-validator" exchange
func (rh *RabbitMQHandler) SendWaggleMessageOnNode(message *datatype.WaggleMessage, scope string) error {
	logger.Debug.Println(string(datatype.Dump(message)))
	return rh.publish(*NewRabbitMQMessageWrapper(
		"to-validator",
		scope,
		datatype.Dump(message),
		"",
	))
}

// SendWaggleMessageOnNodeAsync caches message internally. The background routine will push messages asynchronously.
func (rh *RabbitMQHandler) SendWaggleMessageOnNodeAsync(message *datatype.WaggleMessage, scope string) error {
	if len(rh.chanToPublish) == cap(rh.chanToPublish) {
		return fmt.Errorf("maximum capacity (%d) reached. this message will not be cached", cap(rh.chanToPublish))
	}
	rh.chanToPublish <- *NewRabbitMQMessageWrapper(
		"to-validator",
		scope,
		datatype.Dump(message),
		"",
	)
	return nil
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

// SubscribeEvents subscribes scheduling events from target exchange
// it will attempt to reconnect if connection is closed
func (rh *RabbitMQHandler) SubscribeEvents(exchange string, queueName string, topic string, ch chan datatype.Event) error {
	operation := func() error {
		q, err := rh.DeclareQueueAndConnectToExchange(exchange, queueName, topic)
		if err != nil {
			return err
		}
		c, err := rh.GetReceiver(q.Name)
		if err != nil {
			return err
		}
		for msg := range c {
			if waggleMessage, err := datatype.Load(msg.Body); err == nil {
				eventBuilder, err := datatype.NewSchedulerEventBuilderFromWaggleMessage(waggleMessage)
				if err != nil {
					logger.Debug.Printf("Failed to parse %v: %s", waggleMessage, err.Error())
				} else {
					if vsn, exist := waggleMessage.Meta["vsn"]; exist {
						eventBuilder.AddEntry("vsn", vsn)
					}
					event := eventBuilder.Build()
					ch <- event
				}
			}
		}
		return nil
	}
	go func() {
		for {
			err := backoff.Retry(operation, backoff.NewExponentialBackOff())
			if err != nil {
				logger.Error.Printf("Failed to subscribe %q: %s", exchange, err.Error())
			} else {
				logger.Info.Printf("Connection to %q is closed", exchange)
			}
			logger.Info.Printf("Retrying to connect to %q in 5 seconds...", exchange)
			time.Sleep(5 * time.Second)
		}

	}()
	return nil
}

func (rh *RabbitMQHandler) StartLoop() {
	go func() {
		for m := range rh.chanToPublish {
			logger.Debug.Printf("to %s with routing key %s: %s", m.DestName, m.RoutingKey, m.Body)
			if err := rh.publish(m); err != nil {
				logger.Error.Printf("failed to send message to %s: %s", m.DestName, err.Error())
			}
		}
	}()
}
