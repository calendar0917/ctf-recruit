package admin

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	repo                 Repository
	now                  func() time.Time
	attachmentStorageDir string
}

func NewService(repo Repository, attachmentStorageDir string) *Service {
	return &Service{repo: repo, now: time.Now, attachmentStorageDir: attachmentStorageDir}
}

func (s *Service) Challenges(ctx context.Context) ([]ChallengeSummary, error) {
	return s.repo.ListChallenges(ctx)
}

func (s *Service) Challenge(ctx context.Context, challengeID int64) (ChallengeDetail, error) {
	return s.repo.GetChallenge(ctx, challengeID)
}

func (s *Service) CreateChallenge(ctx context.Context, input UpsertChallengeInput) (ChallengeSummary, error) {
	return s.repo.CreateChallenge(ctx, input)
}

func (s *Service) UpdateChallenge(ctx context.Context, challengeID int64, input UpsertChallengeInput) (ChallengeSummary, error) {
	return s.repo.UpdateChallenge(ctx, challengeID, input)
}

func (s *Service) CreateAttachment(ctx context.Context, actorUserID int64, challengeID int64, input CreateAttachmentInput) (Attachment, error) {
	storagePath, err := s.writeAttachmentFile(challengeID, input.Filename, input.Body)
	if err != nil {
		return Attachment{}, err
	}
	attachment, err := s.repo.CreateAttachment(ctx, challengeID, input.Filename, storagePath, input.ContentType, input.SizeBytes)
	if err != nil {
		_ = os.Remove(storagePath)
		return Attachment{}, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actorUserID, "attachment.create", "challenge_attachment", fmt.Sprintf("%d", attachment.ID), map[string]any{
		"challenge_id": challengeID,
		"filename":     attachment.Filename,
	})
	return attachment, nil
}

func (s *Service) Attachment(ctx context.Context, challengeID int64, attachmentID int64) (Attachment, string, error) {
	return s.repo.GetAttachment(ctx, challengeID, attachmentID)
}

func (s *Service) Users(ctx context.Context) ([]UserRecord, error) {
	return s.repo.ListUsers(ctx)
}

func (s *Service) UpdateUser(ctx context.Context, actorUserID int64, userID int64, input UpdateUserInput) (UserRecord, error) {
	user, err := s.repo.UpdateUser(ctx, userID, input)
	if err != nil {
		return UserRecord{}, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actorUserID, "user.update", "user", fmt.Sprintf("%d", userID), map[string]any{
		"role":         input.Role,
		"display_name": input.DisplayName,
		"status":       input.Status,
	})
	return user, nil
}

func (s *Service) AuditLogs(ctx context.Context) ([]AuditLogRecord, error) {
	return s.repo.ListAuditLogs(ctx)
}

func (s *Service) Announcements(ctx context.Context) ([]Announcement, error) {
	return s.repo.ListAnnouncements(ctx)
}

func (s *Service) CreateAnnouncement(ctx context.Context, actorUserID int64, input CreateAnnouncementInput) (Announcement, error) {
	announcement, err := s.repo.CreateAnnouncement(ctx, actorUserID, input)
	if err != nil {
		return Announcement{}, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actorUserID, "announcement.create", "announcement", fmt.Sprintf("%d", announcement.ID), map[string]any{
		"title":     announcement.Title,
		"published": announcement.Published,
	})
	return announcement, nil
}

func (s *Service) DeleteAnnouncement(ctx context.Context, actorUserID int64, announcementID int64) (Announcement, error) {
	announcement, err := s.repo.DeleteAnnouncement(ctx, announcementID)
	if err != nil {
		return Announcement{}, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actorUserID, "announcement.delete", "announcement", fmt.Sprintf("%d", announcementID), map[string]any{
		"title":     announcement.Title,
		"published": announcement.Published,
	})
	return announcement, nil
}

func (s *Service) Submissions(ctx context.Context) ([]SubmissionRecord, error) {
	return s.repo.ListSubmissions(ctx)
}

func (s *Service) Instances(ctx context.Context) ([]InstanceRecord, error) {
	return s.repo.ListInstances(ctx)
}

func (s *Service) TerminateInstance(ctx context.Context, actorUserID int64, instanceID int64) (InstanceRecord, error) {
	instance, err := s.repo.TerminateInstance(ctx, instanceID, s.now().UTC())
	if err != nil {
		return InstanceRecord{}, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actorUserID, "instance.terminate", "instance", fmt.Sprintf("%d", instanceID), map[string]any{
		"challenge_id": instance.ChallengeID,
		"username":     instance.Username,
	})
	return instance, nil
}

func (s *Service) writeAttachmentFile(challengeID int64, filename string, body io.Reader) (string, error) {
	dir := filepath.Join(s.attachmentStorageDir, fmt.Sprintf("challenge-%d", challengeID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create attachment dir: %w", err)
	}
	safeName := sanitizeFilename(filename)
	path := filepath.Join(dir, fmt.Sprintf("%d-%s", s.now().UTC().UnixNano(), safeName))
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create attachment file: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, body); err != nil {
		return "", fmt.Errorf("write attachment file: %w", err)
	}
	return path, nil
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "..", "")
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "attachment.bin"
	}
	return name
}
