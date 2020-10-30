package nodescheduler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

var (
	host         string
	conn         *amqp.Connection
	exchangeName string = "messages"
)

type RMQMessage struct {
	Timestamp int64             `json:"ts"`
	Value     float64           `json:"value"`
	Topic     string            `json:"topic"`
	Scope     string            `json:"scope"`
	Tags      map[string]string `json:"tags"`
}

func InitializeMeasureCollector(hostUrl string) {
	host = hostUrl
}

func RunMeasureCollector(toKnowledgebase chan RMQMessage) {
	for {
		InfoLogger.Print("Measure collector (re)starts...")
		c, err := getConnection()
		if err != nil {
			ErrorLogger.Print(err.Error())
			continue
		}

		ch, err := c.Channel()
		if err != nil {
			ErrorLogger.Print(err.Error())
			continue
		}

		q, err := ch.QueueDeclare(
			"",    // name
			false, // durable
			false, // delete when unused
			true,  // exclusive
			false, // no-wait
			nil,   // arguments
		)

		err = ch.QueueBind(
			q.Name,       // queue name
			"*",          // routing key
			exchangeName, // exchange
			false,
			nil)

		// Start subscription on everything
		msgs, err := ch.Consume(
			q.Name, // queue name
			"",     // consumer
			true,   // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)

		closeNotifyChan := ch.NotifyClose(make(chan *amqp.Error))

		go func() {
			for msg := range msgs {
				// TODO: should drop messages going to Beehive
				InfoLogger.Printf("%s received", msg.Body)
				InfoLogger.Printf("%s", msg.RoutingKey)
				var rmqMessage RMQMessage
				json.Unmarshal(msg.Body, &rmqMessage)
				InfoLogger.Printf("%v", rmqMessage)
				// TODO: We want to filter out ones going to Beehive
				// TODO: We should do the filtering by setting a proper routingkey
				if rmqMessage.Scope == "node" {
					toKnowledgebase <- rmqMessage
				}
			}
		}()

		err = <-closeNotifyChan
		ErrorLogger.Print(err.Error())
		InfoLogger.Print("Measure collector restarting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func getCredential(host string, id string, pw string) string {
	return fmt.Sprintf("amqp://%s:%s@%s", id, pw, host)
}

func getConnection() (*amqp.Connection, error) {
	if conn == nil || conn.IsClosed() {
		return amqp.Dial(getCredential(host, "worker", "worker"))
	}
	return conn, nil
}
