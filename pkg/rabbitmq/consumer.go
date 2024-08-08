package rabbitmq

import (
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitClient struct {
	conn                      *amqp.Connection
	consumeChan               *amqp.Channel
	pushChan                  *amqp.Channel
	timeout                   time.Duration
	pushQueue                 string
	uri                       string
	l                         *slog.Logger
	notifyConnectionClose     chan *amqp.Error
	notifyConsumeChannelClose chan *amqp.Error
	notifyPushChannelClose    chan *amqp.Error
}

func InitRabbitClient(uri, pushQueue string, timeout time.Duration, l *slog.Logger) (*RabbitClient, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}
	consumeChan, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	pushChan, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return &RabbitClient{
		conn:        conn,
		consumeChan: consumeChan,
		pushChan:    pushChan,
		pushQueue:   pushQueue,
		timeout:     timeout,
		uri:         uri,
		l:           l,
	}, nil
}

func (rc *RabbitClient) GracefulShutdown() {
	rc.consumeChan.Close()
	rc.pushChan.Close()
	rc.conn.Close()
}

// func (rc *RabbitClient) PushSession(session sessions.Session) error {
// 	// TODO: временная заглушка. Костя попросил.
// 	if rc.pushQueue == "" {
// 		return nil
// 	}
// 	b, err := json.Marshal(session)
// 	if err != nil {
// 		return err
// 	}
// 	ctx, cancel := context.WithTimeout(context.Background(), rc.timeout)
// 	defer cancel()
// 	return rc.pushChan.PublishWithContext(
// 		ctx,
// 		"",
// 		rc.pushQueue,
// 		false,
// 		false,
// 		amqp.Publishing{
// 			ContentType: "appplication/json",
// 			Body:        b,
// 		})
// }

func (rc *RabbitClient) startListenRabbitMQ(queue string, c chan<- []byte) {
	_, err := rc.consumeChan.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		amqp.Table{"x-queue-type": "quorum"},
	)
	if err != nil {
		rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "consumeChan.QueueDeclare", "error", err)
		return
	}

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
		case err := <-rc.notifyPushChannelClose:
			rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "rc.notifyPushChannelClose", "error", err)
			return
		case d := <-msgs:
			c <- d.Body
		}
	}
}

func (rc *RabbitClient) StartListenRabbitMQ(queue string, c chan<- []byte) {
	for {
		rc.notifyConnectionClose = rc.conn.NotifyClose(make(chan *amqp.Error))
		rc.notifyConsumeChannelClose = rc.consumeChan.NotifyClose(make(chan *amqp.Error))
		rc.notifyPushChannelClose = rc.pushChan.NotifyClose(make(chan *amqp.Error))
		rc.startListenRabbitMQ(queue, c)
		time.Sleep(time.Second * 5) // 5 секунд перерыв, перед попыткой подключиться к RabbitMQ, ведь он не поднимется моментально
		rc.l.Error("RabbitClient - StartListenRabbitMQ - restart connection")

		for {
			conn, err := amqp.Dial(rc.uri)
			if err == nil {
				rc.conn = conn
				break
			}
			rc.l.Error("RabbitClient - StartListenRabbitMQ - amqp.Dial", "error", err)
		}
		for {
			consumeChan, err := rc.conn.Channel()
			if err == nil {
				rc.consumeChan = consumeChan
				break
			}
			rc.l.Error("RabbitClient - StartListenRabbitMQ - conn.Channel for consumeChan", "error", err)
		}
		for {
			pushChan, err := rc.conn.Channel()
			if err == nil {
				rc.pushChan = pushChan
				break
			}
			rc.l.Error("RabbitClient - StartListenRabbitMQ - conn.Channel for pushChan", "error", err)
		}
	}
}
