package app

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"ctf/backend/internal/admin"
	"ctf/backend/internal/challengeimport"
	"ctf/backend/internal/config"
	"ctf/backend/internal/httpx"
	"ctf/backend/internal/store"
)

type adminImportChallengesRequest struct {
	ContestSlug   string `json:"contest_slug"`
	Root          string `json:"root"`
	Path          string `json:"path"`
	AttachmentDir string `json:"attachment_dir"`
}

type adminImportChallengesResult struct {
	Imported int      `json:"imported"`
	Slugs    []string `json:"slugs"`
}

// handleAdminImportChallenges imports challenge.yaml files from a server-local path.
// This is intentionally development-oriented; production deployments should restrict filesystem access.
func (s *Server) handleAdminImportChallenges(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := userIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}
	actor, ok := adminActorFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated user")
		return
	}

	var input adminImportChallengesRequest
	if err := decodeJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	contestSlug := strings.TrimSpace(input.ContestSlug)
	if contestSlug == "" {
		contestSlug = "recruit-2025"
	}

	attachmentDir := strings.TrimSpace(input.AttachmentDir)
	if attachmentDir == "" {
		attachmentDir = s.cfg.AttachmentStorageDir
	}

	root := strings.TrimSpace(input.Root)
	if root == "" {
		root = "./challenges"
	}
	root, _ = filepath.Abs(root)

	specPath := strings.TrimSpace(input.Path)
	if specPath != "" {
		specPath, _ = filepath.Abs(specPath)
	}

	// Importer uses its own DB connection.
	cfg := config.Load()
	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "repository_error", "failed to open database")
		return
	}
	defer db.Close()

	importer := challengeimport.NewWithAttachmentStorage(db, attachmentDir)
	ctx := context.Background()

	paths := make([]string, 0)
	if specPath != "" {
		paths = append(paths, specPath)
	} else {
		discovered, err := challengeimport.DiscoverSpecFiles(root)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_root", err.Error())
			return
		}
		paths = discovered
	}
	if len(paths) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "no_specs", "no challenge.yaml specs found")
		return
	}

	imported := make([]string, 0, len(paths))
	for _, p := range paths {
		result, err := importer.ImportFile(ctx, contestSlug, p)
		if err != nil {
			if errors.Is(err, admin.ErrResourceNotFound) {
				httpx.WriteError(w, http.StatusNotFound, "resource_not_found", err.Error())
				return
			}
			httpx.WriteError(w, http.StatusBadRequest, "import_failed", err.Error())
			return
		}
		imported = append(imported, result.Slug)
	}

	// admin.Service currently doesn't expose a direct audit write helper.
	// Persist the audit log via repository for now.
	adminRepo := store.NewAdminRepository(s.db)
	_ = adminRepo.CreateAuditLog(r.Context(), &actorUserID, "challenge.import", "challenge", contestSlug, map[string]any{
		"count": len(imported),
		"root":  root,
		"path":  specPath,
		"slugs": imported,
		"actor": actor.Role,
	})

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"result": adminImportChallengesResult{Imported: len(imported), Slugs: imported}})
}
