package ws

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

type Consumer struct {
	Ch                 chan<- []byte
	AllowedOrigins     []string
	muxer              http.ServeMux
	Server             http.Server
	Pattern            string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	ServerPort         int
	InsecureSkipVerify bool
}

func (c *Consumer) Connect() error {
	c.muxer.HandleFunc(c.Pattern, c.handler)
	c.Server = http.Server{
		Handler:      c,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,
	}
	li, err := net.Listen("tcp", fmt.Sprintf(":%d", c.ServerPort))
	if err != nil {
		return err
	}

	go c.Server.Serve(li)
	return nil
}

func (c *Consumer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.muxer.ServeHTTP(w, r)
}

func (c *Consumer) handler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: c.InsecureSkipVerify,
		OriginPatterns:     c.AllowedOrigins,
	})
	if err != nil {
		return
	}
	// TODO: think about this
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()
	// err = conn.Ping(ctx)
	// if err != nil {
	// c.l.Error("Server - wsHandler - Ping", "error", err)
	// 				cancel()
	// 				cs.errorProcessing(user_id, c)
	// 				return
	// }
	for {
		msg_t, msg, err := conn.Read(context.TODO())
		if err != nil {
			// cs.logger.Error("Server - wsHandler - Read", "error", err)
			return
		}
		_ = msg_t // json могут отправлять, а он будет текстовым тогда
		// поэтому бы не проверял на бинарный тип
		// if msg_t == websocket.MessageBinary {
		c.Ch <- msg
		// }
	}
}

func (c *Consumer) GracefulShutdown() {
	c.Server.Close()
}
