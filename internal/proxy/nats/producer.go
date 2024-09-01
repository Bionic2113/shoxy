package nats

import "github.com/Bionic2113/shoxy/pkg/nats"

type ProducerConfig struct {
	Url             string
	Queue           string
	Stream          string
	Subjects        []string
	FailedSize      int64
	Name            string
	RetentionPolicy int
}

type producer struct {
	p *nats.Producer
}

// TODO: default config
func NewProducer(ch <-chan []byte) *producer {
	return &producer{}
}

func (p *producer) WithUrl(url string) *producer {
	p.p.Url = url
	return p
}

func (p *producer) WithQueue(queue string) *producer {
	p.p.Queue = queue
	return p
}

// func (c *producer) WithName(name string) *producer {
// 	c.p.Name = name
// 	return c
// }

func (p *producer) WithStream(stream string) *producer {
	p.p.Stream = stream
	return p
}

func (p *producer) WithFailedSize(fs int64) *producer {
	p.p.Failed = make([][]byte, 0, fs)
	return p
}

func (p *producer) WithRetentionPolicy(rp int) *producer {
	p.p.RetentionPolicy = rp
	return p
}

func (p *producer) WithSubjects(subs []string) *producer {
	p.p.Subjects = subs
	return p
}
