package broker

import (
	"context"
	"github.com/nats-io/nats.go"
	"log/slog"
)

type Publisher struct {
	log     *slog.Logger
	nc      *nats.Conn
	subject string
}

func NewPublisher(log *slog.Logger, addr, subject string) (*Publisher, error) {
	nc, err := nats.Connect(addr)
	if err != nil {
		return nil, err
	}
	log.Info("connected to broker", "addr", addr, "subject", subject)

	return &Publisher{
		log:     log,
		nc:      nc,
		subject: subject,
	}, nil
}

func (p *Publisher) Close() {
	p.nc.Close()
}

func (p *Publisher) NotifyDBUpdated(ctx context.Context) {
	if err := p.nc.Publish(p.subject, []byte("XKCD DB has been updated")); err != nil {
		p.log.Error("Failed to publish updated data", "error", err)
		return
	}
	if err := p.nc.Flush(); err != nil {
		p.log.Error("could not publish message", "error", err)
	}
	p.log.Info("db updated event published")
}
