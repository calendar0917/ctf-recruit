INSERT INTO roles (name)
VALUES ('ops'), ('author')
ON CONFLICT (name) DO NOTHING;
