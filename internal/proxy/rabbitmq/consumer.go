package rabbitmq

import (
	"time"

	"github.com/Bionic2113/shoxy/pkg/rabbitmq"
)

type ConsumerConfig struct {
	SendChan       chan<- []byte
	Url            string
	Queue          string
	RecoverTimeout time.Duration
	XQueueType     string
}

type consumer struct {
	c rabbitmq.Consumer
}

// TODO: default config
func NewConsumer(ch chan<- []byte) *consumer {
	return &consumer{}
}

func (c *consumer) WithUrl(url string) *consumer {
	c.c.Url = url
	return c
}

func (c *consumer) WithQueue(q string) *consumer {
	c.c.Queue = q
	return c
}

func (c *consumer) WithRecoverTimeout(t time.Duration) *consumer {
	c.c.RecoverTimeout = t
	return c
}

func (c *consumer) WithXQueueType(xqt string) *consumer {
	c.c.XQueueType = xqt
	return c
}
