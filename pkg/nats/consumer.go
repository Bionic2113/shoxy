package nats

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Consumer struct {
	Conn           *nats.Conn
	JS             jetstream.JetStream
	log            *slog.Logger
	Cctx           jetstream.ConsumeContext
	MaxDeliver     int
	BackOff        []time.Duration
	Url            string
	Queue          string
	Group          string
	Name           string
	Description    string
	Stream         string
	FilterSubjects []string
	Chan           chan<- []byte
}

func (c *Consumer) Connect() (err error) {
	c.Conn, err = nats.Connect(c.Url)
	if err != nil {
		return
	}
	c.JS, err = jetstream.New(c.Conn)
	return
}

func (c *Consumer) Consume(ctx context.Context) error {
	consumer, err := c.JS.CreateOrUpdateConsumer(context.TODO(), c.Stream, jetstream.ConsumerConfig{
		Name:           c.Name,
		Durable:        c.Name,
		Description:    c.Description,
		BackOff:        c.BackOff,
		MaxDeliver:     c.MaxDeliver,
		FilterSubjects: c.FilterSubjects,
	})
	if err != nil {
		return err
	}
	c.Cctx, err = consumer.Consume(func(msg jetstream.Msg) {
		c.Chan <- msg.Data()
		msg.Ack()
	})
	return err
}

func (c *Consumer) GracefulShutdown() {
	c.Cctx.Stop()
	c.Conn.Drain()
	c.Conn.Close()
}
