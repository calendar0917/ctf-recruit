package admin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"ctf/backend/internal/game"
)

type fakeRepo struct {
	users                 []UserRecord
	auditLogs             []AuditLogRecord
	instances             []InstanceRecord
	createdChallengeInput UpsertChallengeInput
	createdChallengeActor Actor
	updatedChallengeInput UpsertChallengeInput
	updatedChallengeActor Actor
	updatedAuthorUserIDs  []int64
	attachmentActor       Actor
}

type fakeManager struct {
	stopped []string
	err     error
}

func (m *fakeManager) Stop(_ context.Context, containerID string) error {
	if m.err != nil {
		return m.err
	}
	m.stopped = append(m.stopped, containerID)
	return nil
}

func (r *fakeRepo) ListChallenges(context.Context, Actor) ([]ChallengeSummary, error) {
	return []ChallengeSummary{{ID: 1, Slug: "web-welcome"}}, nil
}

func (r *fakeRepo) GetChallenge(context.Context, Actor, int64) (ChallengeDetail, error) {
	return ChallengeDetail{ID: 1, Slug: "web-welcome", RuntimeConfig: RuntimeConfig{Enabled: true, ImageName: "ctf/web-welcome:dev"}}, nil
}

func (r *fakeRepo) CreateChallenge(_ context.Context, actor Actor, input UpsertChallengeInput) (ChallengeSummary, error) {
	r.createdChallengeActor = actor
	r.createdChallengeInput = input
	return ChallengeSummary{ID: 2, Slug: input.Slug}, nil
}

func (r *fakeRepo) UpdateChallenge(_ context.Context, actor Actor, _ int64, input UpsertChallengeInput) (ChallengeSummary, error) {
	r.updatedChallengeActor = actor
	r.updatedChallengeInput = input
	return ChallengeSummary{ID: 1, Slug: input.Slug}, nil
}

func (r *fakeRepo) ListChallengeAuthors(context.Context, Actor, int64) ([]ChallengeAuthor, error) {
	return []ChallengeAuthor{{UserID: 7, Username: "author", Role: "author"}}, nil
}

func (r *fakeRepo) UpdateChallengeAuthors(_ context.Context, _ Actor, _ int64, userIDs []int64) ([]ChallengeAuthor, error) {
	r.updatedAuthorUserIDs = append([]int64(nil), userIDs...)
	items := make([]ChallengeAuthor, 0, len(userIDs))
	for _, userID := range userIDs {
		items = append(items, ChallengeAuthor{UserID: userID, Username: fmt.Sprintf("user-%d", userID), Role: "author"})
	}
	return items, nil
}

func (r *fakeRepo) CreateAttachment(_ context.Context, actor Actor, _ int64, filename, _, contentType string, sizeBytes int64) (Attachment, error) {
	r.attachmentActor = actor
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

func (r *fakeRepo) DeleteAnnouncement(context.Context, int64) (Announcement, error) {
	return Announcement{ID: 1, Title: "hello", Published: true}, nil
}

func (r *fakeRepo) ListSubmissions(context.Context) ([]SubmissionRecord, error) {
	return []SubmissionRecord{{ID: 1}}, nil
}

func (r *fakeRepo) ListInstances(context.Context) ([]InstanceRecord, error) {
	return r.instances, nil
}

func (r *fakeRepo) GetInstance(_ context.Context, instanceID int64) (InstanceRecord, error) {
	for _, item := range r.instances {
		if item.ID == instanceID {
			return item, nil
		}
	}
	return InstanceRecord{}, ErrResourceNotFound
}

func (r *fakeRepo) TerminateInstance(_ context.Context, instanceID int64, terminatedAt time.Time) (InstanceRecord, error) {
	for i := range r.instances {
		if r.instances[i].ID == instanceID {
			r.instances[i].Status = "terminated"
			t := terminatedAt.UTC()
			r.instances[i].TerminatedAt = &t
			return r.instances[i], nil
		}
	}
	return InstanceRecord{}, ErrResourceNotFound
}

func TestChallenge(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())
	challenge, err := service.Challenge(context.Background(), Actor{UserID: 1, Role: "admin"}, 1)
	if err != nil {
		t.Fatalf("challenge: %v", err)
	}
	if challenge.RuntimeConfig.ImageName != "ctf/web-welcome:dev" {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
}

func TestCreateChallengeNormalizesSupportedFlagType(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())
	actor := Actor{UserID: 7, Role: "author"}

	_, err := service.CreateChallenge(context.Background(), actor, UpsertChallengeInput{
		Slug:         "welcome",
		Title:        "Welcome",
		CategorySlug: "web",
		Difficulty:   "easy",
		FlagType:     " CASE_INSENSITIVE ",
		FlagValue:    "Flag{Welcome}",
	})
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	if repo.createdChallengeInput.FlagType != game.FlagTypeCaseInsensitive {
		t.Fatalf("expected normalized flag type, got %q", repo.createdChallengeInput.FlagType)
	}
	if repo.createdChallengeActor != actor {
		t.Fatalf("expected actor to be forwarded, got %+v", repo.createdChallengeActor)
	}
}

func TestCreateChallengeNormalizesPublishedStatusFromLegacyVisible(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())

	_, err := service.CreateChallenge(context.Background(), Actor{UserID: 1, Role: "admin"}, UpsertChallengeInput{
		Slug:         "welcome",
		Title:        "Welcome",
		CategorySlug: "web",
		Difficulty:   "easy",
		FlagType:     game.FlagTypeStatic,
		FlagValue:    "flag{welcome}",
		Visible:      true,
	})
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	if repo.createdChallengeInput.Status != "published" {
		t.Fatalf("expected published status, got %q", repo.createdChallengeInput.Status)
	}
	if !repo.createdChallengeInput.Visible {
		t.Fatalf("expected visible compatibility flag to remain true")
	}
}

func TestCreateChallengeRejectsInvalidStatus(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())

	_, err := service.CreateChallenge(context.Background(), Actor{UserID: 1, Role: "admin"}, UpsertChallengeInput{
		Slug:         "welcome",
		Title:        "Welcome",
		CategorySlug: "web",
		Difficulty:   "easy",
		FlagType:     game.FlagTypeStatic,
		FlagValue:    "flag{welcome}",
		Status:       "launching",
	})
	if !errors.Is(err, ErrInvalidChallengeInput) {
		t.Fatalf("expected invalid challenge input, got %v", err)
	}
}

func TestCreateChallengeRejectsInvalidRegexFlagType(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())

	_, err := service.CreateChallenge(context.Background(), Actor{UserID: 1, Role: "admin"}, UpsertChallengeInput{
		Slug:         "welcome",
		Title:        "Welcome",
		CategorySlug: "web",
		Difficulty:   "easy",
		FlagType:     game.FlagTypeRegex,
		FlagValue:    "^(broken$",
	})
	if !errors.Is(err, ErrInvalidChallengeInput) {
		t.Fatalf("expected invalid challenge input, got %v", err)
	}
}

func TestUpdateChallengeAuthorsDeduplicatesAndAudits(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())
	actor := Actor{UserID: 1, Role: "admin"}
	authors, err := service.UpdateChallengeAuthors(context.Background(), actor, 5, UpdateChallengeAuthorsInput{UserIDs: []int64{7, 7, 8, 0}})
	if err != nil {
		t.Fatalf("update challenge authors: %v", err)
	}
	if !reflect.DeepEqual(repo.updatedAuthorUserIDs, []int64{7, 8}) {
		t.Fatalf("unexpected author ids: %+v", repo.updatedAuthorUserIDs)
	}
	if len(authors) != 2 {
		t.Fatalf("expected 2 authors, got %+v", authors)
	}
	if len(repo.auditLogs) != 1 || repo.auditLogs[0].Action != "challenge.authors.update" {
		t.Fatalf("expected challenge authors audit log, got %+v", repo.auditLogs)
	}
}

func TestCreateAttachmentWritesFileAndAudit(t *testing.T) {
	repo := &fakeRepo{}
	storageDir := t.TempDir()
	service := NewService(repo, storageDir)
	service.now = func() time.Time { return time.Date(2025, time.March, 8, 12, 0, 0, 0, time.UTC) }
	actor := Actor{UserID: 9, Role: "author"}

	attachment, err := service.CreateAttachment(context.Background(), actor, 1, CreateAttachmentInput{
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
	if repo.attachmentActor != actor {
		t.Fatalf("expected actor to be forwarded, got %+v", repo.attachmentActor)
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

func TestTerminateInstanceStopsContainerAndAudits(t *testing.T) {
	repo := &fakeRepo{instances: []InstanceRecord{{ID: 1, ChallengeID: 7, Username: "alice", Status: "running", ContainerID: "cid-1"}}}
	manager := &fakeManager{}
	service := NewServiceWithManager(repo, t.TempDir(), manager)
	instance, err := service.TerminateInstance(context.Background(), 2, 1)
	if err != nil {
		t.Fatalf("terminate instance: %v", err)
	}
	if instance.Status != "terminated" {
		t.Fatalf("unexpected status: %s", instance.Status)
	}
	if len(manager.stopped) != 1 || manager.stopped[0] != "cid-1" {
		t.Fatalf("expected container stop, got %+v", manager.stopped)
	}
	if len(repo.auditLogs) != 1 || repo.auditLogs[0].Action != "instance.terminate" {
		t.Fatalf("expected terminate audit log, got %+v", repo.auditLogs)
	}
}

func TestTerminateInstanceReturnsStopFailure(t *testing.T) {
	repo := &fakeRepo{instances: []InstanceRecord{{ID: 1, ChallengeID: 7, Username: "alice", Status: "running", ContainerID: "cid-1"}}}
	manager := &fakeManager{err: errors.New("stop failed")}
	service := NewServiceWithManager(repo, t.TempDir(), manager)
	_, err := service.TerminateInstance(context.Background(), 2, 1)
	if err == nil || err.Error() != "stop failed" {
		t.Fatalf("expected stop failure, got %v", err)
	}
	if len(repo.auditLogs) != 0 {
		t.Fatalf("unexpected audit logs on failure: %+v", repo.auditLogs)
	}
}

func TestDeleteAnnouncement(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo, t.TempDir())
	announcement, err := service.DeleteAnnouncement(context.Background(), 2, 1)
	if err != nil {
		t.Fatalf("delete announcement: %v", err)
	}
	if announcement.ID != 1 {
		t.Fatalf("unexpected announcement: %+v", announcement)
	}
	if len(repo.auditLogs) != 1 || repo.auditLogs[0].Action != "announcement.delete" {
		t.Fatalf("expected delete audit log, got %+v", repo.auditLogs)
	}
}
