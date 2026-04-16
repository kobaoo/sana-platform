-- ==========================================
-- SEED: users
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
) VALUES
('11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'kc-user-1', 'admin@sana.kz', 'ADMIN', true, true, NOW(), NOW()),
('22222222-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'kc-user-2', 'manager@sana.kz', 'MANAGER', true, true, NOW(), NOW()),
('33333333-cccc-cccc-cccc-cccccccccccc', 'kc-user-3', 'user1@sana.kz', 'EMP', true, true, NOW(), NOW()),
('44444444-dddd-dddd-dddd-dddddddddddd', 'kc-user-4', 'user2@sana.kz', 'EMP', true, true, NOW(), NOW()),
('55555555-eeee-eeee-eeee-eeeeeeeeeeee', 'kc-user-5', 'user3@sana.kz', 'EMP', true, true, NOW(), NOW());
