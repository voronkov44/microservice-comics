package core

import (
	"context"
	"sort"
	"strings"
)

const (
	defaultLimit = 10

	weightTitle = 5
	weightAlt   = 3
	weightWords = 1
)

type Service struct {
	db    DB
	words Words

	index *InvertedIndex
}

func NewService(db DB, words Words) *Service {
	return &Service{
		db:    db,
		words: words,

		index: NewInvertedIndex(),
	}
}

// RebuildIndex - вызывается инициатором, полностью пересобирает индекс из БД.
func (s *Service) RebuildIndex(ctx context.Context) error {
	comics, err := s.db.All(ctx)
	if err != nil {
		return err
	}

	s.index.Build(comics)
	return nil
}

func (s *Service) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Service) Find(ctx context.Context, phrase string, limit uint32) ([]Comics, error) {
	phrase = strings.TrimSpace(phrase)
	if phrase == "" {
		return nil, ErrEmptyPhrase
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if limit > 100 {
		return nil, ErrToLargeLimit
	}

	// нормализуем фразу
	tokens, err := s.words.Norm(ctx, phrase)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, ErrNonePhrase
	}

	// получаем кандидатов из бд
	comics, err := s.db.Find(ctx, tokens)
	if err != nil {
		return nil, err
	}

	result, _ := rangComics(comics, tokens, limit)
	return result, nil
}

// IndexedSearch - метод поиска по индексу
func (s *Service) IndexedSearch(ctx context.Context, phrase string, limit uint32) ([]Comics, uint32, error) {
	phrase = strings.TrimSpace(phrase)
	if phrase == "" {
		return nil, 0, ErrEmptyPhrase
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if limit > 100 {
		return nil, 0, ErrToLargeLimit
	}

	// нормализуем фразу
	tokens, err := s.words.Norm(ctx, phrase)
	if err != nil {
		return nil, 0, err
	}
	if len(tokens) == 0 {
		return nil, 0, ErrNonePhrase
	}

	// получаем ID комиксов из индекса
	ids := s.index.DocsForTokens(tokens)
	if len(ids) == 0 {
		return nil, 0, nil
	}

	// берём сами комиксы из памяти
	candidates := s.index.DocsByIDs(ids)

	// ранжируем по тому же алгоритму
	result, total := rangComics(candidates, tokens, limit)
	return result, total, nil
}

// rangComics - общая функция для ранжирования для Find и IndexedSearch
func rangComics(comics []Comics, tokens []string, limit uint32) ([]Comics, uint32) {
	type scored struct {
		comic Comics
		score int
	}

	scoredList := make([]scored, 0, len(comics))
	for _, c := range comics {
		score := scoreComic(c, tokens)
		if score > 0 {
			scoredList = append(scoredList, scored{
				comic: c,
				score: score,
			})
		}
	}

	total := uint32(len(scoredList))

	// сортировка - по score убывает, при равенстве по ID возрастает
	sort.Slice(scoredList, func(i, j int) bool {
		if scoredList[i].score == scoredList[j].score {
			return scoredList[i].comic.ID < scoredList[j].comic.ID
		}
		return scoredList[i].score > scoredList[j].score
	})

	// применяем limit
	if uint32(len(scoredList)) > limit {
		scoredList = scoredList[:limit]
	}

	out := make([]Comics, 0, len(scoredList))
	for _, sc := range scoredList {
		out = append(out, sc.comic)
	}
	return out, total
}

// scoreComic - функция для подсчета весов
func scoreComic(c Comics, tokens []string) int {
	titleSet := makeSet(c.Title)
	altSet := makeSet(c.Alt)
	wordsSet := makeSet(c.Words)

	var titleMatches, altMatches, wordsMatches int

	// coveered - сет, для уникальных токенов, которые встречаются в любом поле
	covered := make(map[string]struct{}, len(tokens))

	for _, t := range tokens {
		matched := false
		if titleSet[t] {
			titleMatches++
			matched = true
		}
		if altSet[t] {
			altMatches++
			matched = true
		}
		if wordsSet[t] {
			wordsMatches++
			matched = true
		}

		if matched {
			covered[t] = struct{}{}
		}
	}

	if len(covered) == 0 {
		return 0
	}

	// coveredTokens нужен чтобы комикс, который покрывает много токенов стоял выше остальных
	coveredTokens := len(covered)

	return coveredTokens*100 + titleMatches*weightTitle + altMatches*weightAlt + wordsMatches*weightWords
}

// Сет для перевода слайса в мапу для более быстрой проверки (O(1) вместо O(n))
func makeSet(arr []string) map[string]bool {
	m := make(map[string]bool, len(arr))
	for _, v := range arr {
		if v == "" {
			continue
		}
		m[v] = true
	}
	return m
}
