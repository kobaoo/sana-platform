-- ==========================================
-- SEED: users (initiator)
-- ==========================================

INSERT INTO users (
    id,
    keycloak_user_id,
    email,
    role,
    is_active,
    is_onboarded,
    created_at,
    updated_at
) VALUES (
    'cccccccc-cccc-cccc-cccc-cccccccccccc',
    'kc-user-1',
    'test@sana.kz',
    'ADMIN',
    true,
    true,
    NOW(),
    NOW()
);