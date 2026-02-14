ALTER TABLE challenges
    ADD COLUMN IF NOT EXISTS mode VARCHAR(20) NOT NULL DEFAULT 'static';

CREATE INDEX IF NOT EXISTS idx_challenges_mode_deleted_at ON challenges (mode, deleted_at);
