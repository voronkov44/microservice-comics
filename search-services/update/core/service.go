package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Service
// Потокобезопасность запуска обеспечивается atomic флагом running
type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	concurrency int

	running atomic.Bool
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		concurrency: concurrency,
	}, nil
}

func (s *Service) Update(ctx context.Context) (err error) {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	defer s.running.Store(false)

	// Ласт айдишник комикса
	latest, err := s.xkcd.LastID(ctx)
	if err != nil {
		return err
	}

	// Вызываем метод и получаем слайс уже имеющих айдишников в базе
	have, err := s.db.IDs(ctx)
	if err != nil {
		return err
	}
	// Создаем мапу - хеш таблицу
	// struct{} - потому что bool - занимает 1 байт, а структура - ничего, значение нам неважно
	// len(have) -  выделяем емкость под число элементов в слайсе
	exists := make(map[int]struct{}, len(have))
	for _, id := range have {
		exists[id] = struct{}{}
	}

	workers := s.concurrency
	if workers > 64 {
		workers = 64
	}

	// Создаем буфферизированный канал, емкостью в 2 воркера - для отправки немного задач вперед, пока воркеры отдыхают
	// 2 воркера - отличное значение, не слишком большое (иначе съест память) и не слишком маленькое (иначе будет блокироваться main)
	type job struct{ id int }
	jobs := make(chan job, workers*2)

	var wg sync.WaitGroup

	worker := func() {
		for {
			select {
			case <-ctx.Done():
				return // быстрый выход по отмене
			case j, ok := <-jobs:
				if !ok {
					return // канал закрыт - работа закончена
				}

				// Загружаем коммиксы с xkcd
				info, err := s.xkcd.Get(ctx, j.id)
				if err != nil {
					// если ошибка - спокойно пропускаем, так как не все номера существуют (404),
					// но добавляем в базу номер комикса и пустые значения
					if errors.Is(err, ErrNotFound) {
						_ = s.db.Add(ctx, Comics{
							ID:    j.id,
							URL:   "",
							Title: []string{},
							Alt:   []string{},
							Words: []string{},
						})
						continue
					}
					s.log.Warn("xkcd get failed", "id", j.id, "err", err)
					continue
				}

				// Нормализация
				title, errTitle := s.words.Norm(ctx, info.Title)
				if errTitle != nil {
					s.log.Warn("normalize title failed, storing empty", "id", j.id, "err", errTitle)
					title = []string{}
				}

				alt, errAlt := s.words.Norm(ctx, info.Alt)
				if errAlt != nil {
					s.log.Warn("normalize alt failed, storing empty", "id", j.id, "err", errAlt)
					alt = []string{}
				}

				words, errDesc := s.words.Norm(ctx, info.Description)
				if errDesc != nil {
					s.log.Warn("normalize description failed, storing empty words", "id", j.id, "err", errDesc)
					words = []string{}
				}

				if err := s.db.Add(ctx, Comics{
					ID:    info.ID,
					URL:   info.URL,
					Title: title,
					Alt:   alt,
					Words: words,
				}); err != nil {
					s.log.Warn("db add failed", "id", j.id, "err", err)
					continue
				}
			}
		}
	}

	// Запуск воркеров
	for i := 0; i < workers; i++ {
		wg.Go(worker)
	}
	// Отправляем задачи - все id от 1 до latest, кроме существующих
	for id := 1; id <= latest; id++ {
		if _, ok := exists[id]; ok {
			continue
		}
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		case jobs <- job{id: id}:
		}
	}
	close(jobs)
	wg.Wait()

	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbst, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, err
	}
	total, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, err
	}
	return ServiceStats{
		DBStats:     dbst,
		ComicsTotal: total,
	}, nil
}

func (s *Service) Status(_ context.Context) ServiceStatus {
	if s.running.Load() {
		return StatusRunning
	}
	return StatusIdle
}

func (s *Service) Drop(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	defer s.running.Store(false)

	return s.db.Drop(ctx)
}
