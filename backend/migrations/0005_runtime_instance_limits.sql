ALTER TABLE challenge_runtime_configs
    ADD COLUMN IF NOT EXISTS max_active_instances INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS user_cooldown_seconds INT NOT NULL DEFAULT 0;
