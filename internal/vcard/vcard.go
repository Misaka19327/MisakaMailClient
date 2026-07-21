// Package vcard parses vCard (3.0/4.0) text into minimal Card values. It
// handles only the fields relevant to contacts (FN, N, EMAIL) and folded
// continuation lines.
package vcard

import (
	"strings"
)

// Card is a parsed vCard with the fields we care about.
type Card struct {
	Name   string
	Emails []string
}

// Parse parses one or more vCards from data.
func Parse(data string) []Card {
	// Unfold: a line beginning with space or tab continues the previous line
	// (the leading space is dropped, the rest appended).
	var lines []string
	for _, raw := range strings.Split(data, "\n") {
		raw = strings.TrimRight(raw, "\r")
		if len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t') && len(lines) > 0 {
			lines[len(lines)-1] += raw[1:]
			continue
		}
		lines = append(lines, raw)
	}

	var cards []Card
	var cur *Card
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "BEGIN:VCARD") {
			cur = &Card{}
			continue
		}
		if strings.EqualFold(line, "END:VCARD") {
			if cur != nil {
				cards = append(cards, *cur)
				cur = nil
			}
			continue
		}
		if cur == nil {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		keyPart := line[:idx]
		value := line[idx+1:]
		key := keyPart
		if i := strings.Index(keyPart, ";"); i >= 0 {
			key = keyPart[:i]
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		setField(cur, key, value)
	}
	if cur != nil {
		cards = append(cards, *cur)
	}
	return cards
}

func setField(c *Card, key, value string) {
	switch key {
	case "FN":
		if c.Name == "" {
			c.Name = strings.TrimSpace(value)
		}
	case "N":
		// N:Last;First;Middle;Prefix;Suffix
		if c.Name == "" {
			parts := strings.Split(value, ";")
			if len(parts) >= 2 {
				c.Name = strings.TrimSpace(parts[1] + " " + parts[0])
			} else if len(parts) == 1 {
				c.Name = strings.TrimSpace(parts[0])
			}
		}
	case "EMAIL":
		if v := strings.TrimSpace(value); v != "" {
			c.Emails = append(c.Emails, v)
		}
	}
}
