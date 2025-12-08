package words

type set map[string]bool

func newSet(capHint int) set {
	if capHint < 0 {
		capHint = 0
	}
	return make(map[string]bool, capHint)
}

func (s set) Add(word string) bool {
	if s[word] {
		return false
	}
	s[word] = true
	return true
}

func (s set) Has(word string) bool {
	return s[word]
}
