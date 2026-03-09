package contest

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrContestNotFound = errors.New("contest not found")

const (
	StatusDraft    = "draft"
	StatusUpcoming = "upcoming"
	StatusRunning  = "running"
	StatusFrozen   = "frozen"
	StatusEnded    = "ended"
)

type Contest struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	StartsAt    *time.Time `json:"starts_at,omitempty"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
}

type Phase struct {
	Status                 string     `json:"status"`
	AnnouncementVisible    bool       `json:"announcement_visible"`
	ChallengeListVisible   bool       `json:"challenge_list_visible"`
	ChallengeDetailVisible bool       `json:"challenge_detail_visible"`
	AttachmentVisible      bool       `json:"attachment_visible"`
	ScoreboardVisible      bool       `json:"scoreboard_visible"`
	SubmissionAllowed      bool       `json:"submission_allowed"`
	RuntimeAllowed         bool       `json:"runtime_allowed"`
	RegistrationAllowed    bool       `json:"registration_allowed"`
	StartsAt               *time.Time `json:"starts_at,omitempty"`
	EndsAt                 *time.Time `json:"ends_at,omitempty"`
	Message                string     `json:"message"`
}

type UpdateInput struct {
	Status   string `json:"status"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

type Repository interface {
	Current(context.Context) (Contest, error)
	Update(context.Context, UpdateInput) (Contest, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Current(ctx context.Context) (Contest, error) {
	current, err := s.repo.Current(ctx)
	if err != nil {
		return Contest{}, err
	}
	current.Status = NormalizeStatus(current.Status)
	return current, nil
}

func (s *Service) Phase(ctx context.Context) (Phase, error) {
	current, err := s.Current(ctx)
	if err != nil {
		return Phase{}, err
	}
	return BuildPhase(current), nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Contest, error) {
	current, err := s.repo.Update(ctx, input)
	if err != nil {
		return Contest{}, err
	}
	current.Status = NormalizeStatus(current.Status)
	return current, nil
}

func BuildPhase(current Contest) Phase {
	status := NormalizeStatus(current.Status)
	phase := Phase{
		Status:    status,
		StartsAt:  current.StartsAt,
		EndsAt:    current.EndsAt,
		Message:   phaseMessage(status),
	}

	switch status {
	case StatusRunning:
		phase.AnnouncementVisible = true
		phase.ChallengeListVisible = true
		phase.ChallengeDetailVisible = true
		phase.AttachmentVisible = true
		phase.ScoreboardVisible = true
		phase.SubmissionAllowed = true
		phase.RuntimeAllowed = true
		phase.RegistrationAllowed = true
	case StatusFrozen:
		phase.AnnouncementVisible = true
		phase.ChallengeListVisible = true
		phase.ChallengeDetailVisible = true
		phase.AttachmentVisible = true
		phase.ScoreboardVisible = true
		phase.SubmissionAllowed = false
		phase.RuntimeAllowed = false
		phase.RegistrationAllowed = true
	case StatusEnded:
		phase.AnnouncementVisible = true
		phase.ChallengeListVisible = true
		phase.ChallengeDetailVisible = true
		phase.AttachmentVisible = true
		phase.ScoreboardVisible = true
		phase.SubmissionAllowed = false
		phase.RuntimeAllowed = false
		phase.RegistrationAllowed = false
	case StatusUpcoming:
		phase.AnnouncementVisible = true
		phase.RegistrationAllowed = true
	default:
		phase.Status = StatusDraft
		phase.Message = phaseMessage(StatusDraft)
	}

	return phase
}

func NormalizeStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case StatusDraft:
		return StatusDraft
	case StatusUpcoming:
		return StatusUpcoming
	case StatusRunning:
		return StatusRunning
	case StatusFrozen:
		return StatusFrozen
	case StatusEnded:
		return StatusEnded
	default:
		return StatusDraft
	}
}

func phaseMessage(status string) string {
	switch status {
	case StatusUpcoming:
		return "比赛尚未开始，当前仅开放注册与赛前说明。"
	case StatusRunning:
		return "比赛进行中，题目、提交和排行榜已开放。"
	case StatusFrozen:
		return "比赛已进入冻结阶段，当前暂停提交与实例创建。"
	case StatusEnded:
		return "比赛已结束，当前保留题目与排行榜查看。"
	default:
		return "比赛仍在准备中，公开内容尚未开放。"
	}
}
