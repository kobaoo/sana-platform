-- ==========================================
-- SEED: training_events
-- ==========================================

INSERT INTO training_events (
    id,
    title,
    start_date,
    end_date,
    location_type,
    location_city,
    category_id,
    direction,
    dzo_id,
    participants_count
) VALUES
('aaaa1111-1111-1111-1111-111111111111', 'Go Backend Basics', NOW(), NOW(), 'offline', 'Astana', '66666666-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'IT', '77777777-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 10),
('bbbb2222-2222-2222-2222-222222222222', 'Docker Fundamentals', NOW(), NOW(), 'online', 'Almaty', '66666666-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'DevOps', '77777777-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 15),
('cccc3333-3333-3333-3333-333333333333', 'Kubernetes Intro', NOW(), NOW(), 'offline', 'Astana', '66666666-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'DevOps', '77777777-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 12),
('dddd4444-4444-4444-4444-444444444444', 'React Basics', NOW(), NOW(), 'online', 'Astana', '66666666-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Frontend', '77777777-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 20),
('eeee5555-5555-5555-5555-555555555555', 'System Design', NOW(), NOW(), 'offline', 'Almaty', '66666666-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Architecture', '77777777-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 8);

-- ==========================================
-- SEED: requests
-- ==========================================

INSERT INTO requests (
    id,
    initiator_id,
    entity_id,
    entity_type,
    step,
    created_at,
    status
) VALUES
('aaaa0001-0000-0000-0000-000000000001', '11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaa1111-1111-1111-1111-111111111111', 'training_event', 1, NOW(), 'draft'),
('aaaa0002-0000-0000-0000-000000000002', '22222222-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'bbbb2222-2222-2222-2222-222222222222', 'training_event', 1, NOW(), 'submitted'),
('aaaa0003-0000-0000-0000-000000000003', '11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'cccc3333-3333-3333-3333-333333333333', 'training_event', 2, NOW(), 'approved'),
('aaaa0004-0000-0000-0000-000000000004', '22222222-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'dddd4444-4444-4444-4444-444444444444', 'training_event', 1, NOW(), 'draft'),
('aaaa0005-0000-0000-0000-000000000005', '11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'eeee5555-5555-5555-5555-555555555555', 'training_event', 3, NOW(), 'rejected');



-- ==========================================
-- SEED: training_participants
-- ==========================================

INSERT INTO training_participants (
    id,
    event_id,
    employee_id,
    status
) VALUES
('bbbb0001-0000-0000-0000-000000000001', 'aaaa1111-1111-1111-1111-111111111111', '33333333-cccc-cccc-cccc-cccccccccccc', 'registered'),
('bbbb0002-0000-0000-0000-000000000002', 'aaaa1111-1111-1111-1111-111111111111', '44444444-dddd-dddd-dddd-dddddddddddd', 'completed'),
('bbbb0003-0000-0000-0000-000000000003', 'bbbb2222-2222-2222-2222-222222222222', '33333333-cccc-cccc-cccc-cccccccccccc', 'registered'),
('bbbb0004-0000-0000-0000-000000000004', 'cccc3333-3333-3333-3333-333333333333', '44444444-dddd-dddd-dddd-dddddddddddd', 'completed'),
('bbbb0005-0000-0000-0000-000000000005', 'dddd4444-4444-4444-4444-444444444444', '33333333-cccc-cccc-cccc-cccccccccccc', 'registered');