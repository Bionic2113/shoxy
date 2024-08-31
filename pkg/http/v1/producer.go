package v1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type producer struct {
	client  *http.Client
	readCh  <-chan []byte
	writeCh chan<- []byte
	l       *slog.Logger
	failed  [][]byte
	address string
	timeout time.Duration
	req     *http.Request
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

func NewProducer(ctx context.Context, cfg ProducerConfig, l *slog.Logger) (*producer, error) {
	req, err := http.NewRequest(http.MethodPost, cfg.Address, bytes.NewReader([]byte{}))
	if err != nil {
		return nil, err
	}
	req.Header = cfg.Headers
	for name, value := range cfg.Cookie {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	p := &producer{
		readCh:  cfg.ReadChan,
		writeCh: cfg.WriteChan,
		address: cfg.Address,
		timeout: cfg.Timeout,
		l:       l,
		failed:  make([][]byte, 0, cfg.FailedSize),
		client: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   cfg.Timeout,
		},
		req: req,
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
	p.req = p.req.WithContext(newCtx)
	p.sendFailed(ctx)

	for msg := range p.readCh {
		p.l.Info("HttpProducer - ", "action", "сейчас буду отправлять", "data", string(msg))
		if err := p.send(ctx, msg); err != nil {
			p.l.Error("HttpProducer - Write", "error", err)
			return
		}
		p.l.Info("HttpProducer - ", "action", "отправил", "data", string(msg))
	}
	p.l.Info("HttpProducer - - end")
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

	p.req = p.req.WithContext(newCtx)
	p.req.Body = io.NopCloser(bytes.NewReader(msg))

	resp, err := p.client.Do(p.req)
	if err != nil {
		p.failed = append(p.failed, msg)
		p.l.Error("")
		return err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	if err = resp.Write(&buf); err != nil {
		p.l.Error("")
		return
	}

	p.writeCh <- buf.Bytes()

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
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
}
