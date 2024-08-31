package ws

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/coder/websocket"
)

type producer struct {
	conn    *websocket.Conn
	ch      <-chan []byte
	l       *slog.Logger
	failed  [][]byte
	address string
	timeout time.Duration
}

type ProducerConfig struct {
	Chan       <-chan []byte
	Address    string
	FailedSize int64
	Timeout    time.Duration
}

func NewProducer(ctx context.Context, cfg ProducerConfig, l *slog.Logger) (*producer, error) {
	p := &producer{
		ch:      cfg.Chan,
		address: cfg.Address,
		timeout: cfg.Timeout,
		l:       l,
		failed:  make([][]byte, 0, cfg.FailedSize),
	}
	go p.monitoring(ctx)
	return p, nil
}

func (p *producer) monitoring(ctx context.Context) {
	for {
		p.connect(ctx)
	}
}

func (p *producer) connect(ctx context.Context) {
	newCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	c, _, err := websocket.Dial(newCtx, "ws://"+p.address, nil)
	if err != nil {
		return
		// ...
	}
	p.conn = c

	p.sendFailed(ctx)

	for msg := range p.ch {
		p.l.Info("WsProducer - Start", "action", "сейчас буду отправлять", "data", string(msg))
		if err := p.send(ctx, msg); err != nil {
			p.l.Error("WsProducer - Write", "error", err)
			return
		}
		p.l.Info("WsProducer - Start", "action", "отправил", "data", string(msg))
	}
	p.l.Info("WsProducer - Start - end")
}

func (p *producer) send(ctx context.Context, msg []byte) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			p.l.Warn("")
			err = fmt.Errorf("panic has been caught: %v", rec)
		}
	}()

	newCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	if err = p.conn.Write(newCtx, websocket.MessageBinary, msg); err != nil {
		p.failed = append(p.failed, msg)
	}

	return
}

func (p *producer) sendFailed(ctx context.Context) {
	cp := make([][]byte, len(p.failed))
	copy(cp, p.failed)
	var deleted []int
	for i, v := range cp {
		if err := p.send(ctx, v); err != nil {
			deleted = append(deleted, i)
		}
	}
	for i := len(deleted) - 1; i >= 0; i-- {
		p.failed = append(p.failed[:deleted[i]], p.failed[deleted[i]+1:]...)
	}
}

func (p *producer) GracefulShutdown() {
	if p.conn != nil {
		p.conn.CloseNow()
	}
}
