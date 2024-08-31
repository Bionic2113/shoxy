package v1

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type consumer struct {
	ch             chan<- []byte
	l              *slog.Logger
	allowedOrigins []string
	muxer          http.ServeMux
	server         http.Server
}

type ConsumerConfig struct {
	Chan           chan<- []byte
	AllowedOrigins []string
	Pattern        string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ServerPort     int
}

func NewConsumer(cfg ConsumerConfig, l *slog.Logger) (*consumer, error) {
	c := &consumer{
		ch:             cfg.Chan,
		l:              l,
		allowedOrigins: cfg.AllowedOrigins,
	}
	c.muxer.HandleFunc(cfg.Pattern, c.handler)
	c.server = http.Server{
		Handler:      c,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	li, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.ServerPort))
	if err != nil {
		return nil, err
	}

	go c.server.Serve(li)

	return c, nil
}

func (c *consumer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.muxer.ServeHTTP(w, r)
}

// TODO: what do u do with response?
func (c *consumer) handler(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	r.Write(&buf)
	c.ch <- buf.Bytes()
	w.WriteHeader(http.StatusOK)
}

func (c *consumer) GracefulShutdown() {
	c.server.Close()
}
