package nats

import (
	"log/slog"
	"time"

	"github.com/Bionic2113/shoxy/pkg/nats"
)

type ConsumerConfig struct {
	Url            string
	Queue          string
	Group          string
	Name           string
	Description    string
	Stream         string
	FilterSubjects []string
	Chan           chan<- []byte
	BackOff        []time.Duration
	MaxDeliver     int
}

type consumer struct {
	//*commonProxy
	c *nats.Consumer
}

// TODO: default config
func NewConsumer(ch chan<- []byte, l *slog.Logger) *consumer {
	return &consumer{}
}

func (c *consumer) WithUrl(url string) *consumer {
	c.c.Url = url
	return c
}

func (c *consumer) WithQueue(queue string) *consumer {
	c.c.Queue = queue
	return c
}

func (c *consumer) WithGroup(group string) *consumer {
	c.c.Group = group
	return c
}

func (c *consumer) WithName(name string) *consumer {
	c.c.Name = name
	return c
}

func (c *consumer) WithDescription(desc string) *consumer {
	c.c.Description = desc
	return c
}

func (c *consumer) WithStream(stream string) *consumer {
	c.c.Stream = stream
	return c
}

func (c *consumer) WithFilteredSubjects(fs []string) *consumer {
	c.c.FilterSubjects = fs
	return c
}

func (c *consumer) WithBackOff(bo []time.Duration) *consumer {
	c.c.BackOff = bo
	return c
}

func (c *consumer) WithMaxDeliver(d int) *consumer {
	c.c.MaxDeliver = d
	return c
}
