package rabbitmq

import (
	"context"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type producer struct {
	queue                 string
	l                     *slog.Logger
	conn                  *amqp.Connection
	ch                    *amqp.Channel
	notifyConnectionClose chan *amqp.Error
	notifyChannelClose    chan *amqp.Error
	failed                [][]byte
	pushTimeout           time.Duration
	mx                    sync.Mutex
}

type ProducerConfig struct {
	Url            string
	Queue          string
	FailedSize     int64
	PushTimeout    time.Duration
	RecoverTimeout time.Duration
	CleanFrequency time.Duration
}

func NewProducer(cfg ProducerConfig, l *slog.Logger) (*producer, error) {
	conn, err := amqp.Dial(cfg.Url)
	if err != nil {
		return nil, err
	}
	pushChan, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	p := &producer{
		conn:        conn,
		ch:          pushChan,
		queue:       cfg.Queue,
		pushTimeout: cfg.PushTimeout,
		failed:      make([][]byte, 0, cfg.FailedSize),
		l:           l,
	}
	go p.recover(cfg.Url, cfg.RecoverTimeout)
	go p.cleanFailed(cfg.CleanFrequency)
	return p, nil
}

func (rc *producer) GracefulShutdown() {
	rc.ch.Close()
	rc.conn.Close()
}

func (rc *producer) Publish(data []byte) error {
	if rc.mx.TryLock() {
		defer rc.mx.Unlock()
	}
	ctx, cancel := context.WithTimeout(context.Background(), rc.pushTimeout)
	defer cancel()
	err := rc.ch.PublishWithContext(
		ctx,
		"",
		rc.queue,
		false,
		false,
		amqp.Publishing{
			Body: data,
		})
	if err != nil {
		rc.failed = append(rc.failed, data)
	}
	return err
}

func (rc *producer) monitoring() {
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
		case err := <-rc.notifyChannelClose:
			rc.l.Error("RabbitClient - startListenRabbitMQ - end", "reason", "rc.notifyPushChannelClose", "error", err)
			return
		}
	}
}

func (rc *producer) pushFailed() {
	cp := make([][]byte, len(rc.failed))
	copy(cp, rc.failed)
	var deleted []int
	for i, v := range cp {
		if err := rc.Publish(v); err != nil {
			deleted = append(deleted, i)
		}
	}
	for i := len(deleted) - 1; i >= 0; i-- {
		rc.failed = append(rc.failed[:deleted[i]], rc.failed[deleted[i]+1:]...)
	}
}

func (rc *producer) cleanFailed(cleanFrequency time.Duration) {
	ticker := time.Tick(cleanFrequency)
	for range ticker {
		rc.mx.Lock()
		rc.pushFailed()
		rc.mx.Unlock()
	}
}

func (rc *producer) recover(url string, recoverTimeout time.Duration) {
	for {
		rc.mx.Lock()
		rc.notifyConnectionClose = rc.conn.NotifyClose(make(chan *amqp.Error))
		rc.notifyChannelClose = rc.ch.NotifyClose(make(chan *amqp.Error))
		rc.pushFailed()
		rc.mx.Unlock()

		rc.monitoring()
		time.Sleep(recoverTimeout)
		rc.l.Error("RabbitClient - StartListenRabbitMQ - restart connection")

		for {
			conn, err := amqp.Dial(url)
			if err == nil {
				rc.mx.Lock()
				rc.conn = conn
				rc.mx.Unlock()
				break
			}
			rc.l.Error("RabbitClient - StartListenRabbitMQ - amqp.Dial", "error", err)
			time.Sleep(time.Second)
		}
		for {
			pushChan, err := rc.conn.Channel()
			if err == nil {
				rc.mx.Lock()
				rc.ch = pushChan
				rc.mx.Unlock()
				break
			}
			rc.l.Error("RabbitClient - StartListenRabbitMQ - conn.Channel for pushChan", "error", err)
			time.Sleep(time.Second)
		}
	}
}
