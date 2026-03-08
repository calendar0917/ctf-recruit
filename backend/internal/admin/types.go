package admin

import (
	"context"
	"errors"
	"io"
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

type CreateAttachmentInput struct {
	Filename    string
	ContentType string
	Body        io.Reader
	SizeBytes   int64
}

type UserRecord struct {
	ID          int64      `json:"id"`
	Role        string     `json:"role"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type UpdateUserInput struct {
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type AuditLogRecord struct {
	ID           int64          `json:"id"`
	ActorUserID  *int64         `json:"actor_user_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Details      map[string]any `json:"details"`
	CreatedAt    time.Time      `json:"created_at"`
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

type InstanceManager interface {
	Stop(context.Context, string) error
}

type Repository interface {
	ListChallenges(context.Context) ([]ChallengeSummary, error)
	GetChallenge(context.Context, int64) (ChallengeDetail, error)
	CreateChallenge(context.Context, UpsertChallengeInput) (ChallengeSummary, error)
	UpdateChallenge(context.Context, int64, UpsertChallengeInput) (ChallengeSummary, error)
	CreateAttachment(context.Context, int64, string, string, string, int64) (Attachment, error)
	GetAttachment(context.Context, int64, int64) (Attachment, string, error)
	ListUsers(context.Context) ([]UserRecord, error)
	UpdateUser(context.Context, int64, UpdateUserInput) (UserRecord, error)
	ListAuditLogs(context.Context) ([]AuditLogRecord, error)
	CreateAuditLog(context.Context, *int64, string, string, string, map[string]any) error
	ListAnnouncements(context.Context) ([]Announcement, error)
	CreateAnnouncement(context.Context, int64, CreateAnnouncementInput) (Announcement, error)
	DeleteAnnouncement(context.Context, int64) (Announcement, error)
	ListSubmissions(context.Context) ([]SubmissionRecord, error)
	ListInstances(context.Context) ([]InstanceRecord, error)
	GetInstance(context.Context, int64) (InstanceRecord, error)
	TerminateInstance(context.Context, int64, time.Time) (InstanceRecord, error)
}
