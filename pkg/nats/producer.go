package nats

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type producer struct {
	n      *nats.Conn
	l      *slog.Logger
	queue  string
	failed [][]byte
	mx     sync.Mutex
	js     jetstream.JetStream
}
type ProducerConfig struct {
	Url        string
	Queue      string
	Stream     string
	Subjects   []string
	FailedSize int64
}

func NewProducer(cfg ProducerConfig, l *slog.Logger) (*producer, error) {
	conn, err := nats.Connect(cfg.Url)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(conn, jetstream.WithPublishAsyncErrHandler(func(js jetstream.JetStream, m *nats.Msg, err error) {
		fmt.Println("jetstream errhandler error: ", err)
	}))
	if err != nil {
		return nil, err
	}
	_, err = js.CreateOrUpdateStream(context.TODO(), jetstream.StreamConfig{
		Name:      cfg.Stream,
		Subjects:  cfg.Subjects,
		Retention: jetstream.WorkQueuePolicy,
	})
	if err != nil {
		return nil, err
	}

	p := &producer{
		n:      conn,
		js:     js,
		failed: make([][]byte, 0, cfg.FailedSize),
		queue:  cfg.Queue,
		l:      l,
	}

	conn.SetReconnectHandler(func(c *nats.Conn) {
		cp := make([][]byte, len(p.failed))
		copy(cp, p.failed)
		var deleted []int
		p.mx.Lock()
		for i, v := range cp {
			if err := p.n.Publish(cfg.Queue, v); err != nil {
				deleted = append(deleted, i)
			}
		}
		for i := len(deleted) - 1; i >= 0; i-- {
			p.failed = append(p.failed[:deleted[i]], p.failed[deleted[i]+1:]...)
		}
		p.mx.Unlock()
	})
	return p, nil
}

func (p *producer) Publish(data []byte) error {
	_, err := p.js.Publish(context.TODO(), p.queue, data)
	if err != nil {
		p.l.Error("NatsProducer - Publish", "error", err)
		p.mx.Lock()
		p.failed = append(p.failed, data)
		p.mx.Unlock()
	}
	return err
}

func (p *producer) GracefulShutdown() {
	p.n.Drain()
	p.n.Close()
}
