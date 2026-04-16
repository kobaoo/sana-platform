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
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Go Backend Basics',
    NOW(),
    NOW(),
    'offline',
    'Astana',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    'IT',
    'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
    10
);

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
) VALUES (
    '22222222-2222-2222-2222-222222222222',
    'cccccccc-cccc-cccc-cccc-cccccccccccc',
    '11111111-1111-1111-1111-111111111111',
    'training_event',
    1,
    NOW(),
    'draft'
);

-- ==========================================
-- SEED: training_participants
-- ==========================================

INSERT INTO training_participants (
    id,
    event_id,
    employee_id,
    status
) VALUES 
(
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    'dddddddd-dddd-dddd-dddd-dddddddddddd',
    'registered'
),
(
    '44444444-4444-4444-4444-444444444444',
    '11111111-1111-1111-1111-111111111111',
    'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
    'completed'
);