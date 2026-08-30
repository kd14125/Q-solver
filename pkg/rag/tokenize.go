package rag

import (
	"strings"
	"unicode"
)

func searchTokens(text string) []string {
	text = strings.ToLower(text)
	seen := make(map[string]struct{})
	result := make([]string, 0, 64)
	add := func(token string) {
		token = strings.TrimSpace(token)
		if token == "" {
			return
		}
		if _, exists := seen[token]; exists {
			return
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}

	var latin strings.Builder
	flushLatin := func() {
		if latin.Len() > 0 {
			add(latin.String())
			latin.Reset()
		}
	}
	var han []rune
	flushHan := func() {
		if len(han) == 1 {
			add(string(han))
		}
		for n := 2; n <= 3; n++ {
			for i := 0; i+n <= len(han); i++ {
				add(string(han[i : i+n]))
			}
		}
		han = han[:0]
	}

	for _, r := range []rune(text) {
		switch {
		case unicode.Is(unicode.Han, r):
			flushLatin()
			han = append(han, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '+' || r == '#':
			flushHan()
			latin.WriteRune(r)
		default:
			flushLatin()
			flushHan()
		}
	}
	flushLatin()
	flushHan()
	return result
}

func searchableText(text string) string {
	return strings.Join(searchTokens(text), " ")
}

func ftsQuery(text string) string {
	tokens := searchTokens(text)
	if len(tokens) > 24 {
		tokens = tokens[:24]
	}
	quoted := make([]string, 0, len(tokens))
	for _, token := range tokens {
		quoted = append(quoted, `"`+strings.ReplaceAll(token, `"`, `""`)+`"`)
	}
	return strings.Join(quoted, " OR ")
}
