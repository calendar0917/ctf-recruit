CREATE TABLE IF NOT EXISTS recruitment_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    school VARCHAR(255) NOT NULL,
    grade VARCHAR(100) NOT NULL,
    direction VARCHAR(100) NOT NULL,
    contact VARCHAR(255) NOT NULL,
    bio TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recruitment_submissions_user_id ON recruitment_submissions (user_id);
CREATE INDEX IF NOT EXISTS idx_recruitment_submissions_created_at ON recruitment_submissions (created_at DESC);
