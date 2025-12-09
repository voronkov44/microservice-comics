package words

import (
	"github.com/kljensen/snowball"
	"github.com/kljensen/snowball/english"
	"log"
	"regexp"
	"strings"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func (s *service) Norm(phrase string) ([]string, error) {
	// lowercase + отчистка + пробелы между слов
	lc := strings.ToLower(phrase)
	clean := nonAlphaNum.ReplaceAllString(lc, " ")
	tokens := strings.Fields(clean)

	out := make([]string, 0, len(tokens))
	seen := newSet(len(tokens))

	for _, w := range tokens {
		if w == "" {
			continue
		}

		// если токен — чисто цифры, оставляем как есть (без стоп-слов и стемминга)
		isDigits := true
		for i := 0; i < len(w); i++ {
			if w[i] < '0' || w[i] > '9' {
				isDigits = false
				break
			}
		}
		if isDigits {
			if seen.Add(w) {
				out = append(out, w)
				log.Printf("Norm add_digits: %q", w)
			}
			continue
		}

		// отсеиваем часто употребляемые слова типа of/a/the/, местоимения и глагольные частицы (will).
		if english.IsStopWord(w) {
			log.Printf("Norm drop_stopword: %q", w)
			continue
		}
		// стемминг
		stem, err := snowball.Stem(w, "english", false)
		if err != nil || stem == "" {
			stem = w
		}
		// добавляем без дублей
		if seen.Add(stem) {
			out = append(out, stem)
			log.Printf("Norm add_digits: %q", w)
		}
	}
	log.Printf("Norm done: in=%d out=%d", len(tokens), len(out))
	return out, nil
}
