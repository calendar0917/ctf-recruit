INSERT INTO users (role_id, username, email, password_hash, display_name, status)
SELECT r.id, 'admin', 'admin@ctf.local', '$2a$10$.WK8QQXKsHJGJKti025ByuL0qjfagrCP2EqQ6DHL7S8w1ebJB0Ujy', 'Admin', 'active'
FROM roles r
WHERE r.name = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM users u WHERE u.username = 'admin'
  );

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
  AND NOT EXISTS (
      SELECT 1 FROM announcements a WHERE a.title = 'Welcome to Recruit 2025'
  );
