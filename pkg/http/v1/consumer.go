package v1

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Consumer struct {
	Chan           chan<- []byte
	AllowedOrigins []string
	Muxer          http.ServeMux
	Server         http.Server
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	Pattern        string
	ServerPort     int
}

func (c *Consumer) Connect() error {
	c.Muxer.HandleFunc(c.Pattern, c.handler)
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
	c.Muxer.ServeHTTP(w, r)
}

// TODO: what do u do with response?
func (c *Consumer) handler(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	r.Write(&buf)
	c.Chan <- buf.Bytes()
	w.WriteHeader(http.StatusOK)
}

func (c *Consumer) GracefulShutdown() {
	c.Server.Close()
}
