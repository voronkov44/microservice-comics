package broker

import (
	"context"
	"github.com/nats-io/nats.go"
	"log/slog"
)

type IndexUpdater interface {
	RebuildIndex(ctx context.Context) error
}

type Subscriber struct {
	log     *slog.Logger
	nc      *nats.Conn
	subject string
	service IndexUpdater
}

func NewSubscriber(log *slog.Logger, addr, subject string, service IndexUpdater) (*Subscriber, error) {
	nc, err := nats.Connect(addr)
	if err != nil {
		return nil, err
	}
	log.Info("connected to broker", "addr", addr, "subject", subject)

	return &Subscriber{
		log:     log,
		nc:      nc,
		subject: subject,
		service: service,
	}, nil
}

func (s *Subscriber) Close() {
	s.nc.Close()
}

func (s *Subscriber) Start(ctx context.Context) error {
	ch := make(chan *nats.Msg, 10)

	sub, err := s.nc.ChanSubscribe(s.subject, ch)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if err := sub.Unsubscribe(); err != nil {
				s.log.Error("failed to unsubscribe", "subject", s.subject, "error", err)
			}
			s.log.Info("nats subscriber stopped", "subject", s.subject)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					s.log.Info("nats channel closed", "subject", s.subject)
					return
				}

				s.log.Info("got db updated event, rebuilding index",
					"subject", s.subject,
					"data", msg.Data,
				)

				if err := s.service.RebuildIndex(ctx); err != nil {
					s.log.Error("rebuild index failed", "error", err)
				}
			}
		}
	}()

	return nil
}
