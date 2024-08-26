package nats

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type consumer struct {
	n    *nats.Conn
	l    *slog.Logger
	cctx jetstream.ConsumeContext
}
type ConsumerConfig struct {
	Url            string
	Queue          string
	Group          string
	Name           string
	Description    string
	Stream         string
	FilterSubjects []string
	Chan           chan<- []byte
}

func NewConsumer(cfg ConsumerConfig, l *slog.Logger) (*consumer, error) {
	conn, err := nats.Connect(cfg.Url)
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	consumer_, err := js.CreateOrUpdateConsumer(context.TODO(), cfg.Stream, jetstream.ConsumerConfig{
		Name:        cfg.Name,
		Durable:     cfg.Name,
		Description: cfg.Description,
		// BackOff: []time.Duration{
		// 	5 * time.Second,
		// 	10 * time.Second,
		// 	15 * time.Second,
		// },
		// MaxDeliver:     4,
		FilterSubjects: cfg.FilterSubjects,
	})
	if err != nil {
		return nil, err
	}

	cctx, err := consumer_.Consume(func(msg jetstream.Msg) {
		// meta, err := msg.Metadata()
		// if err != nil {
		// 	return
		// }
		// log.Printf("Received message sequence: %d; data: %s\n", meta.Sequence.Stream, string(msg.Data()))

		cfg.Chan <- msg.Data()

		msg.Ack()
	})
	if err != nil {
		return nil, err
	}

	c := &consumer{
		n:    conn,
		l:    l,
		cctx: cctx,
	}

	return c, nil
}

// func (c *consumer) SubscribeGroup(queue, group string) {
// 	c.n.QueueSubscribe(queue, group, func(msg *nats.Msg) {
// 		c.l.Info("NatsConsumer - QueueSubscribe", "message", string(msg.Data))
// 		c.ch <- msg.Data
// 	})
// }
//
// func (c *consumer) Subscribe(queue string) {
// 	c.n.Subscribe(queue, func(msg *nats.Msg) {
// 		c.l.Info("NatsConsumer - Subscribe", "message", string(msg.Data))
// 		c.ch <- msg.Data
// 	})
// }

func (c *consumer) GracefulShutdown() {
	c.cctx.Stop()
	c.n.Drain()
	c.n.Close()
}
