DROP INDEX IF EXISTS idx_challenges_mode_deleted_at;

ALTER TABLE challenges
    DROP COLUMN IF EXISTS mode;
