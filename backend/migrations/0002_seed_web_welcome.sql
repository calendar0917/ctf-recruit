INSERT INTO challenges (
    contest_id,
    category_id,
    slug,
    title,
    description,
    points,
    difficulty,
    flag_type,
    flag_value,
    dynamic_enabled,
    visible,
    sort_order
)
SELECT
    c.id,
    cat.id,
    'web-welcome',
    'Welcome Panel',
    'A minimal seeded web challenge for local runtime integration.',
    100,
    'easy',
    'static',
    'flag{welcome}',
    TRUE,
    TRUE,
    10
FROM contests c
JOIN categories cat ON cat.slug = 'web'
WHERE c.slug = 'recruit-2025'
  AND NOT EXISTS (
      SELECT 1 FROM challenges existing WHERE existing.slug = 'web-welcome'
  );

INSERT INTO challenge_runtime_configs (
    challenge_id,
    image_name,
    exposed_protocol,
    container_port,
    default_ttl_seconds,
    max_renew_count,
    memory_limit_mb,
    cpu_limit_millicores,
    env_json,
    command_json,
    enabled
)
SELECT
    ch.id,
    'ctf/web-welcome:dev',
    'http',
    80,
    1800,
    1,
    256,
    500,
    '{}'::jsonb,
    '[]'::jsonb,
    TRUE
FROM challenges ch
WHERE ch.slug = 'web-welcome'
  AND NOT EXISTS (
      SELECT 1 FROM challenge_runtime_configs rc WHERE rc.challenge_id = ch.id
  );
