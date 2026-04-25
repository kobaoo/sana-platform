-- Add current authenticated user to the system
INSERT INTO users (
    id,
    keycloak_user_id,
    email,
    role,
    is_active,
    is_onboarded,
    created_at,
    updated_at
) VALUES
(gen_random_uuid(), '416fe4bd-a93a-43d0-a65b-1ce17d5cccf3', 'user@example.com', 'SA', true, true, NOW(), NOW())
ON CONFLICT (keycloak_user_id) DO NOTHING;
