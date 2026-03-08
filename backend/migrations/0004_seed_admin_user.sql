INSERT INTO users (role_id, username, email, password_hash, display_name, status)
SELECT r.id, 'admin', 'admin@ctf.local', '$2a$10$jABjY8w6vwxKF1u9IiMaI.A99ZTSdKCbFf8gDuwZQzQ0imeISFRCW', 'Admin', 'active'
FROM roles r
WHERE r.name = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM users u WHERE u.username = 'admin'
  );
