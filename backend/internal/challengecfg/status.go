package challengecfg

import (
	"fmt"
	"strings"
)

const (
	StatusDraft     = "draft"
	StatusReview    = "review"
	StatusReady     = "ready"
	StatusPublished = "published"
)

func ParseStatus(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case StatusDraft:
		return StatusDraft, true
	case StatusReview:
		return StatusReview, true
	case StatusReady:
		return StatusReady, true
	case StatusPublished:
		return StatusPublished, true
	default:
		return "", false
	}
}

func NormalizeStatus(value string) string {
	if normalized, ok := ParseStatus(value); ok {
		return normalized
	}
	return StatusDraft
}

func NormalizeInputStatus(status string, legacyVisible bool) (string, error) {
	if strings.TrimSpace(status) == "" {
		if legacyVisible {
			return StatusPublished, nil
		}
		return StatusDraft, nil
	}
	normalized, ok := ParseStatus(status)
	if !ok {
		return "", fmt.Errorf("invalid challenge status %q", status)
	}
	return normalized, nil
}

func IsPublished(status string) bool {
	return NormalizeStatus(status) == StatusPublished
}
