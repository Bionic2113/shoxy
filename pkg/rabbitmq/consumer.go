package rabbitmq

import (
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	notifyConnectionClose     chan *amqp.Error
	notifyConsumeChannelClose chan *amqp.Error
	conn                      *amqp.Connection
	consumeChan               *amqp.Channel
	l                         *slog.Logger
	Url                       string
	XQueueType                string
	RecoverTimeout            time.Duration
	SendChan                  chan<- []byte
	Queue                     string
}

func (c *Consumer) Connect() error {
	conn, err := amqp.Dial(c.Url)
	if err != nil {
		return err
	}
	c.conn = conn
	consumeChan, err := conn.Channel()
	if err != nil {
		return err
	}
	c.consumeChan = consumeChan
	go c.StartListenRabbitMQ(c.Queue, c.SendChan, c.RecoverTimeout)
	return nil
}

func (rc *Consumer) GracefulShutdown() {
	rc.consumeChan.Close()
	rc.conn.Close()
}

func (rc *Consumer) startListenRabbitMQ(queue string, c chan<- []byte) {
	rc.l.Info("startListenRabbitMQ - start")
	_, err := rc.consumeChan.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		amqp.Table{"x-queue-type": rc.XQueueType},
	)
	if err != nil {
		rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "consumeChan.QueueDeclare", "error", err)
		return
	}

	rc.l.Info("startListenRabbitMQ - after QueueDeclare")
	msgs, err := rc.consumeChan.Consume(
		queue, // queue
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "consumeChan.Consume", "error", err)
		return
	}
	rc.l.Info("RabbitClient - startListenRabbitMQ - consume")
	for {
		select {
		case <-time.After(time.Second):
			if rc.conn.IsClosed() {
				rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "conn.IsClosed")
				return
			}
		case err := <-rc.notifyConnectionClose:
			rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "rc.notifyConnectionClose", "error", err)
			return
		case err := <-rc.notifyConsumeChannelClose:
			rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "rc.notifyConsumeChannelClose", "error", err)
			return
		case d := <-msgs:
			c <- d.Body
		}
	}
}

func (rc *Consumer) StartListenRabbitMQ(queue string, c chan<- []byte, recoverTimeout time.Duration) {
	for {
		rc.l.Info("StartListenRabbitMQ - start")
		rc.notifyConnectionClose = rc.conn.NotifyClose(make(chan *amqp.Error))
		rc.notifyConsumeChannelClose = rc.consumeChan.NotifyClose(make(chan *amqp.Error))
		rc.l.Info("StartListenRabbitMQ - call startListenRabbitMQ")
		rc.startListenRabbitMQ(queue, c)
		time.Sleep(recoverTimeout)
		rc.l.Error("RabbitClient - StartListenRabbitMQ - restart connection")

		for {
			conn, err := amqp.Dial(rc.Url)
			if err == nil {
				rc.conn = conn
				break
			}
			rc.l.Error("RabbitClient - StartListenRabbitMQ - amqp.Dial", "error", err)
			time.Sleep(time.Second)
		}
		for {
			consumeChan, err := rc.conn.Channel()
			if err == nil {
				rc.consumeChan = consumeChan
				break
			}
			rc.l.Error("RabbitClient - StartListenRabbitMQ - conn.Channel for consumeChan", "error", err)
			time.Sleep(time.Second)
		}
	}
}
