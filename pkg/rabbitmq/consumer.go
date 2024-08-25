package rabbitmq

import (
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type consumer struct {
	notifyConnectionClose     chan *amqp.Error
	notifyConsumeChannelClose chan *amqp.Error
	conn                      *amqp.Connection
	consumeChan               *amqp.Channel
	l                         *slog.Logger
	uri                       string
	XQueueType                string
}

type ConsumerConfig struct {
	SendChan       chan<- []byte
	Url            string
	Queue          string
	RecoverTimeout time.Duration
	XQueueType     string
}

func NewConsumer(cfg ConsumerConfig, l *slog.Logger) (*consumer, error) {
	conn, err := amqp.Dial(cfg.Url)
	if err != nil {
		return nil, err
	}
	l.Info("NewConsumer - after Dial")
	consumeChan, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	l.Info("NewConsumer - after Channel")
	c := &consumer{
		conn:        conn,
		consumeChan: consumeChan,
		l:           l,
		XQueueType:  cfg.XQueueType,
	}
	go c.StartListenRabbitMQ(cfg.Queue, cfg.SendChan, cfg.RecoverTimeout)
	l.Info("NewConsumer - after StartListenRabbitMQ")
	return c, nil
}

func (rc *consumer) GracefulShutdown() {
	rc.consumeChan.Close()
	rc.conn.Close()
}

func (rc *consumer) startListenRabbitMQ(queue string, c chan<- []byte) {
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

func (rc *consumer) StartListenRabbitMQ(queue string, c chan<- []byte, recoverTimeout time.Duration) {
	for {
		rc.l.Info("StartListenRabbitMQ - start")
		rc.notifyConnectionClose = rc.conn.NotifyClose(make(chan *amqp.Error))
		rc.notifyConsumeChannelClose = rc.consumeChan.NotifyClose(make(chan *amqp.Error))
		rc.l.Info("StartListenRabbitMQ - call startListenRabbitMQ")
		rc.startListenRabbitMQ(queue, c)
		time.Sleep(recoverTimeout)
		rc.l.Error("RabbitClient - StartListenRabbitMQ - restart connection")

		for {
			conn, err := amqp.Dial(rc.uri)
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
