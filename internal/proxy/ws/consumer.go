package ws

import (
	"log/slog"
	"time"

	"github.com/Bionic2113/shoxy/pkg/ws"
)

type ConsumerConfig struct {
	Chan           chan<- []byte
	AllowedOrigins []string
	Pattern        string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ServerPort     int
}

type consumer struct {
	c ws.Consumer
}

// TODO: default config
func NewConsumer(ch chan<- []byte, l *slog.Logger) *consumer {
	return &consumer{}
}

func (c *consumer) WithAllowOrigins(ao []string) *consumer {
	c.c.InsecureSkipVerify = false
	c.c.AllowedOrigins = ao
	return c
}

func (c *consumer) WithPattern(p string) *consumer {
	c.c.Pattern = p
	return c
}

func (c *consumer) WithReadTimeout(t time.Duration) *consumer {
	c.c.ReadTimeout = t
	return c
}

func (c *consumer) WithWriteTimeout(t time.Duration) *consumer {
	c.c.WriteTimeout = t
	return c
}

func (c *consumer) WithServerPort(p int) *consumer {
	c.c.ServerPort = p
	return c
}
