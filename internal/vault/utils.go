package vault

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func generateID() string {
	return uuid.NewString()
}

func normalizeTags(tags []string) []string {
	m := make(map[string]struct{})
	var result []string

	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := m[t]; !ok {
			m[t] = struct{}{}
			result = append(result, t)
		}
	}
	return result
}

func now() int64 {
	return time.Now().Unix()
}
