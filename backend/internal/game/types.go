package game

import (
	"context"
	"errors"
	"time"
)

var (
	ErrChallengeNotFound  = errors.New("challenge not found")
	ErrAttachmentNotFound = errors.New("challenge attachment not found")
)

type Announcement struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Pinned      bool       `json:"pinned"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type Attachment struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type Challenge struct {
	ID          int64        `json:"id"`
	Slug        string       `json:"slug"`
	Title       string       `json:"title"`
	Category    string       `json:"category"`
	Points      int          `json:"points"`
	Difficulty  string       `json:"difficulty"`
	Description string       `json:"description"`
	Dynamic     bool         `json:"dynamic"`
	Attachments []Attachment `json:"attachments"`
}

type UserSubmission struct {
	ID             int64     `json:"id"`
	ChallengeID    int64     `json:"challenge_id"`
	ChallengeSlug  string    `json:"challenge_slug"`
	ChallengeTitle string    `json:"challenge_title"`
	Category       string    `json:"category"`
	Correct        bool      `json:"correct"`
	SubmittedAt    time.Time `json:"submitted_at"`
	SourceIP       string    `json:"source_ip"`
}

type UserSolve struct {
	ID             int64     `json:"id"`
	ChallengeID    int64     `json:"challenge_id"`
	ChallengeSlug  string    `json:"challenge_slug"`
	ChallengeTitle string    `json:"challenge_title"`
	Category       string    `json:"category"`
	SubmissionID   int64     `json:"submission_id"`
	AwardedPoints  int       `json:"awarded_points"`
	SolvedAt       time.Time `json:"solved_at"`
}

type SubmitResult struct {
	SubmissionID  int64      `json:"submission_id"`
	Correct       bool       `json:"correct"`
	Solved        bool       `json:"solved"`
	Message       string     `json:"message"`
	AwardedPoints int        `json:"awarded_points"`
	SolvedAt      *time.Time `json:"solved_at,omitempty"`
}

type ScoreboardSolve struct {
	ChallengeID    int64      `json:"challenge_id"`
	ChallengeSlug  string     `json:"challenge_slug"`
	ChallengeTitle string     `json:"challenge_title"`
	Category       string     `json:"category"`
	Difficulty     string     `json:"difficulty"`
	AwardedPoints  int        `json:"awarded_points"`
	SolvedAt       time.Time  `json:"solved_at"`
}

type ScoreboardEntry struct {
	Rank        int               `json:"rank"`
	UserID      int64             `json:"user_id"`
	Username    string            `json:"username"`
	DisplayName string            `json:"display_name"`
	Score       int               `json:"score"`
	LastSolveAt *time.Time        `json:"last_solve_at,omitempty"`
	Solves      []ScoreboardSolve `json:"solves"`
}

type Repository interface {
	ListAnnouncements(context.Context) ([]Announcement, error)
	GetChallenge(context.Context, string) (Challenge, string, error)
	GetChallengeAttachment(context.Context, string, int64) (Attachment, string, error)
	CreateSubmission(context.Context, int64, int64, string, bool, string) (int64, time.Time, error)
	HasSolved(context.Context, int64, int64) (bool, error)
	CreateSolve(context.Context, int64, int64, int64, int) (time.Time, error)
	ListUserSubmissions(context.Context, int64) ([]UserSubmission, error)
	ListUserSolves(context.Context, int64) ([]UserSolve, error)
	ListScoreboard(context.Context) ([]ScoreboardEntry, error)
}
