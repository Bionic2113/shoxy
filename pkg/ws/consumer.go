package ws

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"nhooyr.io/websocket"
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

func (c *consumer) handler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: c.allowedOrigins,
	})
	if err != nil {
		return
	}

	// err := c.Ping(ctx)
	// 			if err != nil {
	// 				cs.logger.Error("Server - wsHandler - Ping", "error", err)
	// 				cancel()
	// 				cs.errorProcessing(user_id, c)
	// 				return
	// 			}
	msg_t, msg, err := conn.Read(context.TODO())
	if err != nil {
		// cs.logger.Error("Server - wsHandler - Read", "error", err)
		return
	}
	_ = msg_t // json могут отправлять, а он будет текстовым тогда
	// поэтому бы не проверял на бинарный тип
	// if msg_t == websocket.MessageBinary {
	c.ch <- msg
	// }
}

func (c *consumer) GracefulShutdown() {
	c.server.Close()
}
