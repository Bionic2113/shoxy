package v1

import (
	"log/slog"
	"time"

	v1 "github.com/Bionic2113/shoxy/pkg/http/v1"
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
	c v1.Consumer
}

// TODO: default config
func NewConsumer(ch chan<- []byte, l *slog.Logger) *consumer {
	return &consumer{}
}

func (c *consumer) WithPattern(p string) *consumer {
	c.c.Pattern = p
	return c
}

func (c *consumer) WithAllowedOrigins(origins []string) *consumer {
	c.c.AllowedOrigins = origins
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
