package nats

import (
	"context"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Producer struct {
	Conn            *nats.Conn
	Url             string
	Queue           string
	Failed          [][]byte
	mx              sync.Mutex
	JS              jetstream.JetStream
	RetentionPolicy int
	Stream          string
	Subjects        []string
}

func (p *Producer) Connect(ctx context.Context) (err error) {
	p.Conn, err = nats.Connect(p.Url, nats.ReconnectBufSize(0))
	if err != nil {
		return
	}
	p.JS, err = jetstream.New(p.Conn)
	if err != nil {
		return
	}
	_, err = p.JS.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      p.Stream,
		Subjects:  p.Subjects,
		Retention: jetstream.WorkQueuePolicy,
	})
	if err != nil {
		return
	}
	p.Conn.SetReconnectHandler(func(c *nats.Conn) {
		cp := make([][]byte, len(p.Failed))
		copy(cp, p.Failed)
		var deleted []int
		p.mx.Lock()
		for i, v := range cp {
			if err := p.Conn.Publish(p.Queue, v); err != nil {
				deleted = append(deleted, i)
			}
		}
		for i := len(deleted) - 1; i >= 0; i-- {
			p.Failed = append(p.Failed[:deleted[i]], p.Failed[deleted[i]+1:]...)
		}
		p.mx.Unlock()
	})
	return
}

func (p *Producer) Publish(data []byte) error {
	_, err := p.JS.Publish(context.TODO(), p.Queue, data)
	if err != nil {
		p.mx.Lock()
		p.Failed = append(p.Failed, data)
		p.mx.Unlock()
	}
	return err
}

func (p *Producer) GracefulShutdown() {
	p.Conn.Drain()
	p.Conn.Close()
}
