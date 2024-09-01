package ws

import (
	"time"

	"github.com/Bionic2113/shoxy/pkg/ws"
)

type ProducerConfig struct {
	Chan       <-chan []byte
	Address    string
	FailedSize int64
	Timeout    time.Duration
}
type producer struct {
	p ws.Producer
}

// TODO: default config
func NewProducer(ch <-chan []byte, address string) *producer {
	return &producer{}
}

func (p *producer) WithFailedSize(fs int64) *producer {
	p.p.Failed = make([][]byte, 0, fs)
	return p
}

func (p *producer) WithTimeout(t time.Duration) *producer {
	p.p.Timeout = t
	return p
}
