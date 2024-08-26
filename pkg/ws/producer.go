package ws

import (
	"context"
	"log/slog"

	"github.com/coder/websocket"
)

type producer struct {
	conn *websocket.Conn
	ch   <-chan []byte
	l    *slog.Logger
}

type ProducerConfig struct {
	Chan    <-chan []byte
	Address string
}

func NewProducer(cfg ProducerConfig, l *slog.Logger) (*producer, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	// defer cancel()
	ctx := context.TODO()

	c, _, err := websocket.Dial(ctx, "ws://"+cfg.Address, nil)
	if err != nil {
		return nil, err
		// ...
	}
	return &producer{
		ch:   cfg.Chan,
		conn: c,
		l:    l,
	}, nil
}

func (p *producer) Start() {
	ctx := context.TODO()
	for msg := range p.ch {
		p.l.Info("WsProducer - Start", "action", "сейчас буду отправлять", "data", string(msg))
		if err := p.conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
			p.l.Error("WsProducer - Write", "error", err)
		}
		p.l.Info("WsProducer - Start", "action", "отправил", "data", string(msg))
	}
	p.l.Info("WsProducer - Start - end")
}

func (p *producer) GracefulShutdown() {
	p.conn.CloseNow()
}
