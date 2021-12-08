package utility

import (
	"fmt"
	"strings"
)

func ParseSelector(s string) (map[string]string, error) {
	items := map[string]string{}

	s = strings.TrimSpace(s)
	if s == "" {
		return items, nil
	}

	for _, term := range strings.Split(s, ",") {
		k, v, err := parseSelectorTerm(term)
		if err != nil {
			return items, err
		}
		items[k] = v
	}

	return items, nil
}

func parseSelectorTerm(s string) (string, string, error) {
	fields := strings.Split(s, "=")
	if len(fields) != 2 {
		return "", "", fmt.Errorf("selector terms must have exactly one =")
	}
	return strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]), nil
}
