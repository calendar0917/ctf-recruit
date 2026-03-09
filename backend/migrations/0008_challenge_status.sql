ALTER TABLE challenges
ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'draft';

UPDATE challenges
SET status = CASE
    WHEN visible = TRUE THEN 'published'
    ELSE 'draft'
END
WHERE status IS NULL OR status = '' OR status = 'draft';
