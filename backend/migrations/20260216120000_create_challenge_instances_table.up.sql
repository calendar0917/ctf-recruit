CREATE TABLE IF NOT EXISTS challenge_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    container_id VARCHAR(255),
    started_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    cooldown_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_challenge_instances_user_id ON challenge_instances (user_id);
CREATE INDEX IF NOT EXISTS idx_challenge_instances_challenge_id ON challenge_instances (challenge_id);
CREATE INDEX IF NOT EXISTS idx_challenge_instances_status ON challenge_instances (status);
CREATE INDEX IF NOT EXISTS idx_challenge_instances_user_created_at ON challenge_instances (user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_challenge_instances_active_per_user
    ON challenge_instances (user_id)
    WHERE status IN ('starting', 'running');
