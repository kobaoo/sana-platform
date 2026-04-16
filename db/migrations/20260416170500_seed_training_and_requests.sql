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
('11111111-1111-1111-1111-111111111111', 'Go Backend Basics', NOW(), NOW(), 'offline', 'Astana', 'cat-1', 'IT', 'dzo-1', 10),
('22222222-1111-1111-1111-111111111111', 'Docker Fundamentals', NOW(), NOW(), 'online', 'Almaty', 'cat-2', 'DevOps', 'dzo-2', 15),
('33333333-1111-1111-1111-111111111111', 'Kubernetes Intro', NOW(), NOW(), 'offline', 'Astana', 'cat-3', 'DevOps', 'dzo-1', 12),
('44444444-1111-1111-1111-111111111111', 'React Basics', NOW(), NOW(), 'online', 'Astana', 'cat-4', 'Frontend', 'dzo-3', 20),
('55555555-1111-1111-1111-111111111111', 'System Design', NOW(), NOW(), 'offline', 'Almaty', 'cat-5', 'Architecture', 'dzo-2', 8);



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
('r1111111-1111-1111-1111-111111111111', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111', 'training_event', 1, NOW(), 'draft'),
('r2222222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '22222222-1111-1111-1111-111111111111', 'training_event', 1, NOW(), 'submitted'),
('r3333333-3333-3333-3333-333333333333', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '33333333-1111-1111-1111-111111111111', 'training_event', 2, NOW(), 'approved'),
('r4444444-4444-4444-4444-444444444444', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '44444444-1111-1111-1111-111111111111', 'training_event', 1, NOW(), 'draft'),
('r5555555-5555-5555-5555-555555555555', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '55555555-1111-1111-1111-111111111111', 'training_event', 3, NOW(), 'rejected');



-- ==========================================
-- SEED: training_participants
-- ==========================================

INSERT INTO training_participants (
    id,
    event_id,
    employee_id,
    status
) VALUES
('p1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 'registered'),
('p2222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'completed'),
('p3333333-3333-3333-3333-333333333333', '22222222-1111-1111-1111-111111111111', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 'registered'),
('p4444444-4444-4444-4444-444444444444', '33333333-1111-1111-1111-111111111111', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'completed'),
('p5555555-5555-5555-5555-555555555555', '44444444-1111-1111-1111-111111111111', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 'registered');