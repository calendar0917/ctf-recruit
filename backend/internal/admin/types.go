package admin

import (
	"context"
	"errors"
	"time"
)

var ErrResourceNotFound = errors.New("resource not found")

type RuntimeConfig struct {
	Enabled         bool              `json:"enabled"`
	ImageName       string            `json:"image_name"`
	ExposedProtocol string            `json:"exposed_protocol"`
	ContainerPort   int               `json:"container_port"`
	DefaultTTL      int               `json:"default_ttl_seconds"`
	MaxRenewCount   int               `json:"max_renew_count"`
	MemoryLimitMB   int               `json:"memory_limit_mb"`
	CPUMilli        int               `json:"cpu_limit_millicores"`
	Env             map[string]string `json:"env,omitempty"`
	Command         []string          `json:"command,omitempty"`
}

type Attachment struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type ChallengeSummary struct {
	ID             int64  `json:"id"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Points         int    `json:"points"`
	Visible        bool   `json:"visible"`
	DynamicEnabled bool   `json:"dynamic_enabled"`
}

type ChallengeDetail struct {
	ID             int64         `json:"id"`
	Slug           string        `json:"slug"`
	Title          string        `json:"title"`
	Category       string        `json:"category"`
	Description    string        `json:"description"`
	Points         int           `json:"points"`
	Difficulty     string        `json:"difficulty"`
	FlagType       string        `json:"flag_type"`
	FlagValue      string        `json:"flag_value"`
	Visible        bool          `json:"visible"`
	DynamicEnabled bool          `json:"dynamic_enabled"`
	SortOrder      int           `json:"sort_order"`
	Attachments    []Attachment  `json:"attachments"`
	RuntimeConfig  RuntimeConfig `json:"runtime_config"`
}

type UpsertChallengeInput struct {
	Slug           string         `json:"slug"`
	Title          string         `json:"title"`
	CategorySlug   string         `json:"category_slug"`
	Description    string         `json:"description"`
	Points         int            `json:"points"`
	Difficulty     string         `json:"difficulty"`
	FlagType       string         `json:"flag_type"`
	FlagValue      string         `json:"flag_value"`
	DynamicEnabled bool           `json:"dynamic_enabled"`
	Visible        bool           `json:"visible"`
	SortOrder      int            `json:"sort_order"`
	RuntimeConfig  *RuntimeConfig `json:"runtime_config,omitempty"`
}

type Announcement struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Pinned      bool       `json:"pinned"`
	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type CreateAnnouncementInput struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Pinned    bool   `json:"pinned"`
	Published bool   `json:"published"`
}

type SubmissionRecord struct {
	ID            int64     `json:"id"`
	ChallengeID   int64     `json:"challenge_id"`
	ChallengeSlug string    `json:"challenge_slug"`
	Username      string    `json:"username"`
	Correct       bool      `json:"correct"`
	SubmittedAt   time.Time `json:"submitted_at"`
	SourceIP      string    `json:"source_ip"`
}

type InstanceRecord struct {
	ID            int64      `json:"id"`
	ChallengeID   int64      `json:"challenge_id"`
	ChallengeSlug string     `json:"challenge_slug"`
	Username      string     `json:"username"`
	Status        string     `json:"status"`
	HostPort      int        `json:"host_port"`
	ExpiresAt     time.Time  `json:"expires_at"`
	TerminatedAt  *time.Time `json:"terminated_at,omitempty"`
	ContainerID   string     `json:"container_id"`
}

type Repository interface {
	ListChallenges(context.Context) ([]ChallengeSummary, error)
	GetChallenge(context.Context, int64) (ChallengeDetail, error)
	CreateChallenge(context.Context, UpsertChallengeInput) (ChallengeSummary, error)
	UpdateChallenge(context.Context, int64, UpsertChallengeInput) (ChallengeSummary, error)
	ListAnnouncements(context.Context) ([]Announcement, error)
	CreateAnnouncement(context.Context, int64, CreateAnnouncementInput) (Announcement, error)
	ListSubmissions(context.Context) ([]SubmissionRecord, error)
	ListInstances(context.Context) ([]InstanceRecord, error)
	TerminateInstance(context.Context, int64, time.Time) (InstanceRecord, error)
}
