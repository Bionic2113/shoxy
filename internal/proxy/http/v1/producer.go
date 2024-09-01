package v1

import (
	"net/http"
	"net/url"
	"time"

	v1 "github.com/Bionic2113/shoxy/pkg/http/v1"
)

type ProducerConfig struct {
	ReadChan   <-chan []byte
	WriteChan  chan<- []byte
	Address    string
	FailedSize int64
	Timeout    time.Duration
	Headers    map[string][]string
	Cookie     map[string]string
}

type producer struct {
	p v1.Producer
}

// TODO: default config
func NewProducer(wCh chan<- []byte, rCh <-chan []byte, address string) (*producer, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	return &producer{
		p: v1.Producer{
			Client: &http.Client{
				Transport: http.DefaultTransport,
			},
			Req: &http.Request{
				Method:     http.MethodPost,
				URL:        u,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     make(http.Header),
				Body:       nil,
				Host:       u.Host,
			},
		},
	}, nil
}

func (p *producer) WithHeader(h map[string][]string) *producer {
	p.p.Req.Header = h
	return p
}

func (p *producer) WithCookie(c map[string]string) *producer {
	for name, value := range c {
		p.p.Req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}
	return p
}

func (p *producer) WithFailedSize(fs int64) *producer {
	p.p.Failed = make([][]byte, 0, fs)
	return p
}

func (p *producer) WithTimeout(t time.Duration) *producer {
	p.p.Client.Timeout = t
	return p
}
