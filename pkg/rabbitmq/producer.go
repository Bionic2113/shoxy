package rabbitmq

import (
	"context"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer struct {
	conn                  *amqp.Connection
	ch                    *amqp.Channel
	notifyConnectionClose chan *amqp.Error
	notifyChannelClose    chan *amqp.Error
	Failed                [][]byte
	mx                    sync.Mutex
	Url                   string
	Queue                 string
	PushTimeout           time.Duration
	RecoverTimeout        time.Duration
	CleanFrequency        time.Duration
	l                     *slog.Logger
}

func (p *Producer) Connect() error {
	conn, err := amqp.Dial(p.Url)
	if err != nil {
		return err
	}
	p.conn = conn
	pushChan, err := conn.Channel()
	if err != nil {
		return err
	}
	p.ch = pushChan

	go p.recover(p.Url, p.RecoverTimeout)
	go p.cleanFailed(p.CleanFrequency)
	return nil
}

func (rc *Producer) GracefulShutdown() {
	rc.ch.Close()
	rc.conn.Close()
}

func (rc *Producer) Publish(data []byte) error {
	if rc.mx.TryLock() {
		defer rc.mx.Unlock()
	}
	ctx, cancel := context.WithTimeout(context.Background(), rc.PushTimeout)
	defer cancel()
	err := rc.ch.PublishWithContext(
		ctx,
		"",
		rc.Queue,
		false,
		false,
		amqp.Publishing{
			Body: data,
		})
	if err != nil {
		rc.Failed = append(rc.Failed, data)
	}
	return err
}

func (rc *Producer) monitoring() {
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

func (rc *Producer) pushFailed() {
	cp := make([][]byte, len(rc.Failed))
	copy(cp, rc.Failed)
	var deleted []int
	for i, v := range cp {
		if err := rc.Publish(v); err != nil {
			deleted = append(deleted, i)
		}
	}
	for i := len(deleted) - 1; i >= 0; i-- {
		rc.Failed = append(rc.Failed[:deleted[i]], rc.Failed[deleted[i]+1:]...)
	}
}

func (rc *Producer) cleanFailed(cleanFrequency time.Duration) {
	ticker := time.Tick(cleanFrequency)
	for range ticker {
		rc.mx.Lock()
		rc.pushFailed()
		rc.mx.Unlock()
	}
}

func (rc *Producer) recover(url string, recoverTimeout time.Duration) {
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
