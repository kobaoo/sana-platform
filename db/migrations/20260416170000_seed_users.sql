-- ==========================================
-- SEED: users (initiators)
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
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'kc-user-1', 'admin1@sana.kz', 'ADMIN', true, true, NOW(), NOW()),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'kc-user-2', 'admin2@sana.kz', 'ADMIN', true, true, NOW(), NOW()),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'kc-user-3', 'manager@sana.kz', 'MANAGER', true, true, NOW(), NOW()),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'kc-user-4', 'user1@sana.kz', 'EMP', true, true, NOW(), NOW()),
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'kc-user-5', 'user2@sana.kz', 'EMP', true, true, NOW(), NOW());


