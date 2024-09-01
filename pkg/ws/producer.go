package ws

import (
	"context"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

type Producer struct {
	Conn    *websocket.Conn
	Ch      <-chan []byte
	Failed  [][]byte
	Address string
	Timeout time.Duration
}

func (p *Producer) Connect(ctx context.Context) {
	go p.monitoring(ctx)
}

func (p *Producer) monitoring(ctx context.Context) {
	for {
		p.connect(ctx)
	}
}

func (p *Producer) connect(ctx context.Context) {
	newCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	c, _, err := websocket.Dial(newCtx, "ws://"+p.Address, nil)
	if err != nil {
		return
		// ...
	}
	p.Conn = c

	p.sendFailed(ctx)

	for msg := range p.Ch {
		// p.l.Info("WsProducer - Start", "action", "сейчас буду отправлять", "data", string(msg))
		if err := p.send(ctx, msg); err != nil {
			// p.l.Error("WsProducer - Write", "error", err)
			return
		}
		// p.l.Info("WsProducer - Start", "action", "отправил", "data", string(msg))
	}
	// p.l.Info("WsProducer - Start - end")
}

func (p *Producer) send(ctx context.Context, msg []byte) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			// p.l.Warn("")
			err = fmt.Errorf("panic has been caught: %v", rec)
		}
	}()

	newCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	if err = p.Conn.Write(newCtx, websocket.MessageBinary, msg); err != nil {
		p.Failed = append(p.Failed, msg)
	}

	return
}

func (p *Producer) sendFailed(ctx context.Context) {
	cp := make([][]byte, len(p.Failed))
	copy(cp, p.Failed)
	var deleted []int
	for i, v := range cp {
		if err := p.send(ctx, v); err != nil {
			deleted = append(deleted, i)
		}
	}
	for i := len(deleted) - 1; i >= 0; i-- {
		p.Failed = append(p.Failed[:deleted[i]], p.Failed[deleted[i]+1:]...)
	}
}

func (p *Producer) GracefulShutdown() {
	if p.Conn != nil {
		p.Conn.CloseNow()
	}
}
