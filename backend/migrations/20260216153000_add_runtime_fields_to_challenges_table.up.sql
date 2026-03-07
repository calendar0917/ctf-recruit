ALTER TABLE challenges
    ADD COLUMN IF NOT EXISTS runtime_image VARCHAR(255),
    ADD COLUMN IF NOT EXISTS runtime_command TEXT,
    ADD COLUMN IF NOT EXISTS runtime_exposed_port INTEGER;
