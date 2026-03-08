INSERT INTO announcements (
    contest_id,
    title,
    content,
    pinned,
    published,
    published_at,
    created_by
)
SELECT
    c.id,
    'Welcome to Recruit 2025',
    'The platform is now seeded with the first public announcement.',
    TRUE,
    TRUE,
    NOW(),
    u.id
FROM contests c
JOIN users u ON u.username = 'admin'
WHERE c.slug = 'recruit-2025'
  AND EXISTS (SELECT 1 FROM users WHERE username = 'admin')
  AND NOT EXISTS (
      SELECT 1 FROM announcements a WHERE a.title = 'Welcome to Recruit 2025'
  );
