package core

import (
	"sort"
	"sync"
)

type InvertedIndex struct {
	mu      sync.RWMutex
	byToken map[string][]int
	docs    map[int]Comics
}

func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		byToken: make(map[string][]int),
		docs:    make(map[int]Comics),
	}
}

func (idx *InvertedIndex) Build(comics []Comics) {
	byToken := make(map[string][]int, len(comics)*4)
	docs := make(map[int]Comics, len(comics))

	for _, c := range comics {
		docs[c.ID] = c

		seen := make(map[string]struct{}, len(c.Title)+len(c.Alt)+len(c.Words))
		add := func(tok string) {
			if tok == "" {
				return
			}
			if _, ok := seen[tok]; ok {
				return
			}
			seen[tok] = struct{}{}
			byToken[tok] = append(byToken[tok], c.ID)
		}

		for _, t := range c.Title {
			add(t)
		}
		for _, t := range c.Alt {
			add(t)
		}
		for _, t := range c.Words {
			add(t)
		}
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.byToken = byToken
	idx.docs = docs
}

func (idx *InvertedIndex) DocsForTokens(tokens []string) []int {
	if len(tokens) == 0 {
		return nil
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	set := make(map[int]struct{}, len(tokens)*4)
	for _, tok := range tokens {
		ids := idx.byToken[tok]
		for _, id := range ids {
			set[id] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	res := make([]int, 0, len(set))
	for id := range set {
		res = append(res, id)
	}
	sort.Ints(res)
	return res
}

func (idx *InvertedIndex) DocsByIDs(ids []int) []Comics {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	out := make([]Comics, 0, len(ids))
	for _, id := range ids {
		if c, ok := idx.docs[id]; ok {
			out = append(out, c)
		}
	}
	return out
}
