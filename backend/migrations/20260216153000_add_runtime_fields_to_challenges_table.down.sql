ALTER TABLE challenges
    DROP COLUMN IF EXISTS runtime_exposed_port,
    DROP COLUMN IF EXISTS runtime_command,
    DROP COLUMN IF EXISTS runtime_image;
