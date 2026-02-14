package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	reDateTR = regexp.MustCompile(`^\s*(\d{2})\.(\d{2})\.(\d{4})\b`) // 07.02.2026 ...
	reKcal   = regexp.MustCompile(`(?i)Kalori:\s*([0-9]+)`)
	rePrice  = regexp.MustCompile(`(?i)Fiyat[ıi]:\s*([0-9]+)`)
)

func slugTR(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	replacer := strings.NewReplacer(
		"ç", "c", "Ç", "c",
		"ğ", "g", "Ğ", "g",
		"ı", "i", "I", "i", // dotless i + capital I
		"İ", "i", "i̇", "i", // sometimes comes as i + dot
		"ö", "o", "Ö", "o",
		"ş", "s", "Ş", "s",
		"ü", "u", "Ü", "u",
	)
	s = replacer.Replace(s)

	// Replace separators with underscore
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, "’", "")
	s = strings.ReplaceAll(s, "'", "")

	// Keep only [a-z0-9_]
	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if r == ' ' || r == '_' {
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
		// ignore other punctuation
	}

	out := strings.Trim(b.String(), "_")
	out = regexp.MustCompile(`_+`).ReplaceAllString(out, "_")
	if out == "" {
		out = "item"
	}
	return out
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func parseKcal(s string) (int, bool) {
	m := reKcal.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

func parsePrice(s string) (int, bool) {
	m := rePrice.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

func parseDate(s string) (string, bool) {
	m := reDateTR.FindStringSubmatch(s)
	if len(m) != 4 {
		return "", false
		// dd mm yyyy
	}
	return fmt.Sprintf("%s-%s-%s", m[3], m[2], m[1]), true
}
