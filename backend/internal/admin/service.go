package admin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ctf/backend/internal/challengecfg"
	"ctf/backend/internal/game"
)

type Service struct {
	repo                 Repository
	manager              InstanceManager
	now                  func() time.Time
	attachmentStorageDir string
}

type challengeOwnerChecker interface {
	GetChallenge(context.Context, Actor, int64) (ChallengeDetail, error)
}

func challengeOwnedByUser(ctx context.Context, repo challengeOwnerChecker, challengeID int64, userID int64) (bool, error) {
	_, err := repo.GetChallenge(ctx, Actor{UserID: userID, Role: "author"}, challengeID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrResourceNotFound) {
		return false, nil
	}
	return false, err
}

func NewService(repo Repository, attachmentStorageDir string) *Service {
	return NewServiceWithManager(repo, attachmentStorageDir, nil)
}

func NewServiceWithManager(repo Repository, attachmentStorageDir string, manager InstanceManager) *Service {
	return &Service{repo: repo, manager: manager, now: time.Now, attachmentStorageDir: attachmentStorageDir}
}

func (s *Service) Challenges(ctx context.Context, actor Actor) ([]ChallengeSummary, error) {
	return s.repo.ListChallenges(ctx, actor)
}

func (s *Service) Challenge(ctx context.Context, actor Actor, challengeID int64) (ChallengeDetail, error) {
	return s.repo.GetChallenge(ctx, actor, challengeID)
}

func (s *Service) CreateChallenge(ctx context.Context, actor Actor, input UpsertChallengeInput) (ChallengeSummary, error) {
	normalized, err := game.ValidateFlagTypeConfig(input.FlagType, input.FlagValue)
	if err != nil {
		return ChallengeSummary{}, fmt.Errorf("%w: %v", ErrInvalidChallengeInput, err)
	}
	input.FlagType = normalized
	status, err := challengecfg.NormalizeInputStatus(input.Status, input.Visible)
	if err != nil {
		return ChallengeSummary{}, fmt.Errorf("%w: %v", ErrInvalidChallengeInput, err)
	}
	input.Status = status
	input.Visible = challengecfg.IsPublished(status)
	challenge, err := s.repo.CreateChallenge(ctx, actor, input)
	if err != nil {
		return ChallengeSummary{}, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actor.UserID, "challenge.create", "challenge", fmt.Sprintf("%d", challenge.ID), map[string]any{
		"slug":            challenge.Slug,
		"title":           challenge.Title,
		"category":        challenge.Category,
		"points":          challenge.Points,
		"status":          challenge.Status,
		"published":       challenge.Visible,
		"dynamic_enabled": challenge.DynamicEnabled,
	})
	return challenge, nil
}

func (s *Service) UpdateChallenge(ctx context.Context, actor Actor, challengeID int64, input UpsertChallengeInput) (ChallengeSummary, error) {
	normalized, err := game.ValidateFlagTypeConfig(input.FlagType, input.FlagValue)
	if err != nil {
		return ChallengeSummary{}, fmt.Errorf("%w: %v", ErrInvalidChallengeInput, err)
	}
	input.FlagType = normalized
	status, err := challengecfg.NormalizeInputStatus(input.Status, input.Visible)
	if err != nil {
		return ChallengeSummary{}, fmt.Errorf("%w: %v", ErrInvalidChallengeInput, err)
	}
	input.Status = status
	input.Visible = challengecfg.IsPublished(status)
	previous, err := s.repo.GetChallenge(ctx, actor, challengeID)
	if err != nil {
		return ChallengeSummary{}, err
	}
	challenge, err := s.repo.UpdateChallenge(ctx, actor, challengeID, input)
	if err != nil {
		return ChallengeSummary{}, err
	}
	details := map[string]any{
		"slug":               challenge.Slug,
		"title":              challenge.Title,
		"category":           challenge.Category,
		"points":             challenge.Points,
		"status":             challenge.Status,
		"published":          challenge.Visible,
		"dynamic_enabled":    challenge.DynamicEnabled,
		"previous_status":    previous.Status,
		"previous_published": previous.Visible,
	}
	if previous.Status != challenge.Status {
		details["status_transition"] = fmt.Sprintf("%s->%s", previous.Status, challenge.Status)
	}
	_ = s.repo.CreateAuditLog(ctx, &actor.UserID, "challenge.update", "challenge", fmt.Sprintf("%d", challenge.ID), details)
	return challenge, nil
}

func (s *Service) ChallengeAuthors(ctx context.Context, actor Actor, challengeID int64) ([]ChallengeAuthor, error) {
	return s.repo.ListChallengeAuthors(ctx, actor, challengeID)
}

func (s *Service) UpdateChallengeAuthors(ctx context.Context, actor Actor, challengeID int64, input UpdateChallengeAuthorsInput) ([]ChallengeAuthor, error) {
	userIDs := make([]int64, 0, len(input.UserIDs))
	seen := make(map[int64]struct{}, len(input.UserIDs))
	for _, userID := range input.UserIDs {
		if userID <= 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	authors, err := s.repo.UpdateChallengeAuthors(ctx, actor, challengeID, userIDs)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actor.UserID, "challenge.authors.update", "challenge", fmt.Sprintf("%d", challengeID), map[string]any{
		"challenge_id": challengeID,
		"user_ids":     userIDs,
	})
	return authors, nil
}

func (s *Service) CreateAttachment(ctx context.Context, actor Actor, challengeID int64, input CreateAttachmentInput) (Attachment, error) {
	storagePath, err := s.writeAttachmentFile(challengeID, input.Filename, input.Body)
	if err != nil {
		return Attachment{}, err
	}
	attachment, err := s.repo.CreateAttachment(ctx, actor, challengeID, input.Filename, storagePath, input.ContentType, input.SizeBytes)
	if err != nil {
		_ = os.Remove(storagePath)
		return Attachment{}, err
	}
	_ = s.repo.CreateAuditLog(ctx, &actor.UserID, "attachment.create", "challenge_attachment", fmt.Sprintf("%d", attachment.ID), map[string]any{
		"challenge_id": challengeID,
		"filename":     attachment.Filename,
	})
	return attachment, nil
}

func (s *Service) Attachment(ctx context.Context, actor Actor, challengeID int64, attachmentID int64) (Attachment, string, error) {
	if actor.RestrictToOwnedChallenges() {
		allowed, err := challengeOwnedByUser(ctx, s.repo, challengeID, actor.UserID)
		if err != nil {
			return Attachment{}, "", err
		}
		if !allowed {
			return Attachment{}, "", ErrResourceNotFound
		}
	}
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
	if s.manager != nil {
		current, err := s.repo.GetInstance(ctx, instanceID)
		if err != nil {
			return InstanceRecord{}, err
		}
		if current.ContainerID != "" && current.Status != "terminated" {
			if err := s.manager.Stop(ctx, current.ContainerID); err != nil {
				return InstanceRecord{}, err
			}
		}
	}

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

func IsInvalidChallengeInput(err error) bool {
	if errors.Is(err, ErrInvalidChallengeInput) {
		return true
	}
	return errors.Is(err, game.ErrInvalidFlagStrategy)
}
