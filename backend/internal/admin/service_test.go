package admin

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeRepo struct {
	users     []UserRecord
	auditLogs []AuditLogRecord
}

func (r *fakeRepo) ListChallenges(context.Context) ([]ChallengeSummary, error) {
	return []ChallengeSummary{{ID: 1, Slug: "web-welcome"}}, nil
}
func (r *fakeRepo) GetChallenge(context.Context, int64) (ChallengeDetail, error) {
	return ChallengeDetail{ID: 1, Slug: "web-welcome", RuntimeConfig: RuntimeConfig{Enabled: true, ImageName: "ctf/web-welcome:dev"}}, nil
}
func (r *fakeRepo) CreateChallenge(context.Context, UpsertChallengeInput) (ChallengeSummary, error) {
	return ChallengeSummary{ID: 2, Slug: "new-challenge"}, nil
}
func (r *fakeRepo) UpdateChallenge(context.Context, int64, UpsertChallengeInput) (ChallengeSummary, error) {
	return ChallengeSummary{ID: 1, Slug: "web-welcome"}, nil
}
func (r *fakeRepo) CreateAttachment(_ context.Context, _ int64, filename, _, contentType string, sizeBytes int64) (Attachment, error) {
	return Attachment{ID: 1, Filename: filename, ContentType: contentType, SizeBytes: sizeBytes}, nil
}
func (r *fakeRepo) GetAttachment(context.Context, int64, int64) (Attachment, string, error) {
	return Attachment{ID: 1, Filename: "statement.pdf", ContentType: "application/pdf", SizeBytes: 128}, "/tmp/statement.pdf", nil
}
func (r *fakeRepo) ListUsers(context.Context) ([]UserRecord, error) {
	return r.users, nil
}
func (r *fakeRepo) UpdateUser(_ context.Context, userID int64, input UpdateUserInput) (UserRecord, error) {
	return UserRecord{ID: userID, Role: input.Role, DisplayName: input.DisplayName, Status: input.Status}, nil
}
func (r *fakeRepo) ListAuditLogs(context.Context) ([]AuditLogRecord, error) {
	return r.auditLogs, nil
}
func (r *fakeRepo) CreateAuditLog(_ context.Context, actorUserID *int64, action, resourceType, resourceID string, details map[string]any) error {
	id := int64(len(r.auditLogs) + 1)
	r.auditLogs = append(r.auditLogs, AuditLogRecord{ID: id, ActorUserID: actorUserID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Details: details})
	return nil
}
func (r *fakeRepo) ListAnnouncements(context.Context) ([]Announcement, error) {
	return []Announcement{{ID: 1, Title: "hello"}}, nil
}
func (r *fakeRepo) CreateAnnouncement(context.Context, int64, CreateAnnouncementInput) (Announcement, error) {
	return Announcement{ID: 2, Title: "new"}, nil
}
func (r *fakeRepo) ListSubmissions(context.Context) ([]SubmissionRecord, error) {
	return []SubmissionRecord{{ID: 1}}, nil
}
func (r *fakeRepo) ListInstances(context.Context) ([]InstanceRecord, error) {
	return []InstanceRecord{{ID: 1}}, nil
}
func (r *fakeRepo) TerminateInstance(context.Context, int64, time.Time) (InstanceRecord, error) {
	return InstanceRecord{ID: 1, Status: "terminated"}, nil
}

func TestChallenge(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())
	challenge, err := service.Challenge(context.Background(), 1)
	if err != nil {
		t.Fatalf("challenge: %v", err)
	}
	if challenge.RuntimeConfig.ImageName != "ctf/web-welcome:dev" {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
}

func TestCreateAttachmentWritesFileAndAudit(t *testing.T) {
	repo := &fakeRepo{}
	storageDir := t.TempDir()
	service := NewService(repo, storageDir)
	service.now = func() time.Time { return time.Date(2025, time.March, 8, 12, 0, 0, 0, time.UTC) }

	attachment, err := service.CreateAttachment(context.Background(), 9, 1, CreateAttachmentInput{
		Filename:    "statement.pdf",
		ContentType: "application/pdf",
		Body:        bytes.NewBufferString("pdf-data"),
		SizeBytes:   int64(len("pdf-data")),
	})
	if err != nil {
		t.Fatalf("create attachment: %v", err)
	}
	if attachment.Filename != "statement.pdf" {
		t.Fatalf("unexpected attachment: %+v", attachment)
	}
	matches, err := filepath.Glob(filepath.Join(storageDir, "challenge-1", "*-statement.pdf"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected stored attachment file, got %v %v", matches, err)
	}
	if _, err := os.Stat(matches[0]); err != nil {
		t.Fatalf("stat stored attachment: %v", err)
	}
	if len(repo.auditLogs) != 1 || repo.auditLogs[0].Action != "attachment.create" {
		t.Fatalf("expected attachment audit log, got %+v", repo.auditLogs)
	}
}

func TestUsersAndAuditLogs(t *testing.T) {
	now := time.Date(2025, time.March, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		users:     []UserRecord{{ID: 1, Role: "player", Username: "alice", Status: "active", CreatedAt: now}},
		auditLogs: []AuditLogRecord{{ID: 1, Action: "challenge.update", ResourceType: "challenge", ResourceID: "1", CreatedAt: now}},
	}
	service := NewService(repo, t.TempDir())

	users, err := service.Users(context.Background())
	if err != nil || len(users) != 1 {
		t.Fatalf("users: %v %+v", err, users)
	}
	logs, err := service.AuditLogs(context.Background())
	if err != nil || len(logs) != 1 {
		t.Fatalf("audit logs: %v %+v", err, logs)
	}
}

func TestTerminateInstance(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())
	instance, err := service.TerminateInstance(context.Background(), 2, 1)
	if err != nil {
		t.Fatalf("terminate instance: %v", err)
	}
	if instance.Status != "terminated" {
		t.Fatalf("unexpected status: %s", instance.Status)
	}
	if len(repo.auditLogs) != 1 || repo.auditLogs[0].Action != "instance.terminate" {
		t.Fatalf("expected terminate audit log, got %+v", repo.auditLogs)
	}
}
