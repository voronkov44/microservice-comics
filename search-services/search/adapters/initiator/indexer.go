package initiator

import (
	"context"
	"log/slog"
	"time"
)

type IndexUpdater interface {
	RebuildIndex(ctx context.Context) error
}

type IndexInitiator struct {
	log     *slog.Logger
	service IndexUpdater
	ttl     time.Duration
}

func New(log *slog.Logger, service IndexUpdater, ttl time.Duration) *IndexInitiator {
	return &IndexInitiator{
		log:     log,
		service: service,
		ttl:     ttl,
	}
}

func (i *IndexInitiator) Start(ctx context.Context) {
	go i.loop(ctx)
}

func (i *IndexInitiator) loop(ctx context.Context) {
	if err := i.service.RebuildIndex(ctx); err != nil {
		i.log.Error("initial index build failed", "error", err)
	}

	ticker := time.NewTicker(i.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			i.log.Info("index initiator stopped")
			return
		case <-ticker.C:
			if err := i.service.RebuildIndex(ctx); err != nil {
				i.log.Error("periodic index rebuild failed", "error", err)
			}
		}
	}
}
