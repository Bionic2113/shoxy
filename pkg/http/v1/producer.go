package v1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Producer struct {
	Client  *http.Client
	ReadCh  <-chan []byte
	WriteCh chan<- []byte
	Failed  [][]byte
	Address string
	Timeout time.Duration
	Req     *http.Request
}

type ProducerConfig struct {
	ReadChan   <-chan []byte
	WriteChan  chan<- []byte
	Address    string
	FailedSize int64
	Timeout    time.Duration
	Headers    map[string][]string
	Cookie     map[string]string
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
	p.Req = p.Req.WithContext(newCtx)
	p.sendFailed(ctx)

	for msg := range p.ReadCh {
		if err := p.send(ctx, msg); err != nil {
			return
		}
	}
}

func (p *Producer) send(ctx context.Context, msg []byte) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic has been caught: %v", rec)
		}
	}()

	newCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	p.Req = p.Req.WithContext(newCtx)
	p.Req.Body = io.NopCloser(bytes.NewReader(msg))

	resp, err := p.Client.Do(p.Req)
	if err != nil {
		p.Failed = append(p.Failed, msg)
		return err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	if err = resp.Write(&buf); err != nil {
		return
	}

	p.WriteCh <- buf.Bytes()

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
	if p.Client != nil {
		p.Client.CloseIdleConnections()
	}
}
