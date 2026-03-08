package runtime

import (
	"context"
	"errors"
	"time"
)

var (
	ErrChallengeNotFound         = errors.New("challenge not found")
	ErrChallengeNotDynamic       = errors.New("challenge is not dynamic")
	ErrRuntimeConfigMissing      = errors.New("runtime config missing")
	ErrInstanceNotFound          = errors.New("instance not found")
	ErrInstanceRenewLimitReached = errors.New("instance renew limit reached")
	ErrRepositoryNotFound        = errors.New("repository record not found")
)

type ChallengeConfig struct {
	ID              string
	Slug            string
	Title           string
	Category        string
	Points          int
	Dynamic         bool
	ImageName       string
	ExposedProtocol string
	ContainerPort   int
	TTL             time.Duration
	MaxRenewCount   int
	MemoryLimitMB   int
	CPUMilli        int
	Env             map[string]string
	Command         []string
}

type ChallengeSummary struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Points     int    `json:"points"`
	Difficulty string `json:"difficulty"`
	Dynamic    bool   `json:"dynamic"`
}

type Instance struct {
	ChallengeID   string     `json:"challenge_id"`
	UserID        int64      `json:"user_id,omitempty"`
	Status        string     `json:"status"`
	AccessURL     string     `json:"access_url,omitempty"`
	HostPort      int        `json:"host_port,omitempty"`
	RenewCount    int        `json:"renew_count"`
	StartedAt     time.Time  `json:"started_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	TerminatedAt  *time.Time `json:"terminated_at,omitempty"`
	ContainerID   string     `json:"-"`
	ContainerName string     `json:"-"`
	HostIP        string     `json:"-"`
}

type RuntimeConfigRecord struct {
	ID        int64
	Challenge ChallengeConfig
}

type InstanceRecord struct {
	ID              int64
	RuntimeConfigID int64
	Instance        Instance
}

type Repository interface {
	ListChallenges(context.Context) ([]ChallengeSummary, error)
	GetChallengeConfig(context.Context, string) (RuntimeConfigRecord, error)
	GetActiveInstance(context.Context, int64, string) (InstanceRecord, error)
	CreateInstance(context.Context, int64, Instance) (InstanceRecord, error)
	RenewInstance(context.Context, int64, time.Time) (InstanceRecord, error)
	TerminateInstance(context.Context, int64, time.Time) error
	ListExpiredInstances(context.Context, time.Time) ([]InstanceRecord, error)
}

type StartRequest struct {
	ChallengeID string
	UserID      int64
	Config      ChallengeConfig
}

type StartedContainer struct {
	ContainerID   string
	ContainerName string
	HostIP        string
	HostPort      int
}

type Manager interface {
	Start(context.Context, StartRequest) (StartedContainer, error)
	Stop(context.Context, string) error
}
