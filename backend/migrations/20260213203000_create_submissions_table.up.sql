CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    awarded_points INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_user_id_created_at ON submissions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_submissions_challenge_id_created_at ON submissions (challenge_id, created_at DESC);

-- Prevent double-award while still allowing multiple correct submissions (awarded_points=0)
CREATE UNIQUE INDEX IF NOT EXISTS uq_submissions_award_once ON submissions (user_id, challenge_id) WHERE awarded_points > 0;
