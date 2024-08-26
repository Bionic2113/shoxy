package ws

import (
	"context"
	"log/slog"
	"time"

	"nhooyr.io/websocket"
)

type producer struct {
	conn *websocket.Conn
	ch   <-chan []byte
}

type ProducerConfig struct {
	Chan    <-chan []byte
	Address string
}

func NewProducer(cfg ProducerConfig, l *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := websocket.Dial(ctx, "ws://"+cfg.Address, nil)
	if err != nil {
		// ...
	}

	go func() {
		for msg := range cfg.Chan {
			if err := c.Write(ctx, websocket.MessageBinary, msg); err != nil {
			}
		}
	}()
}

func (p *producer) GracefulShutdown() {
	p.conn.CloseNow()
}
