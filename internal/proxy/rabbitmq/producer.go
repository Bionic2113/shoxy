package rabbitmq

import (
	"time"

	"github.com/Bionic2113/shoxy/pkg/rabbitmq"
)

type ProducerConfig struct {
	Url            string
	Queue          string
	FailedSize     int64
	PushTimeout    time.Duration
	RecoverTimeout time.Duration
	CleanFrequency time.Duration
	Ch             <-chan []byte
}

type producer struct {
	p rabbitmq.Producer
}

// TODO: default config
func NewProducer(ch <-chan []byte) *producer {
	return &producer{}
}

func (p *producer) WithUrl(url string) *producer {
	p.p.Url = url
	return p
}

func (p *producer) WithQueue(q string) *producer {
	p.p.Queue = q
	return p
}

func (p *producer) WithFailedSize(fs int64) *producer {
	p.p.Failed = make([][]byte, 0, fs)
	return p
}

func (p *producer) WithPushTimeout(t time.Duration) *producer {
	p.p.PushTimeout = t
	return p
}

func (p *producer) WithRecoverTimeout(t time.Duration) *producer {
	p.p.RecoverTimeout = t
	return p
}

func (p *producer) WithCleanFrequency(t time.Duration) *producer {
	p.p.CleanFrequency = t
	return p
}
