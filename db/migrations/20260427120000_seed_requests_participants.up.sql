-- Seed request participants and target DZO for existing test requests
-- Adds minimal data so submit/split flows have employees and DZO targets

INSERT INTO public.request_participants (
  id, request_id, employee_id, created_at
) VALUES
(
  'c0ffee00-0000-0000-0000-000000000001',
  'aaaa0001-0000-0000-0000-000000000001',
  '66666666-6666-6666-6666-666666666666',
  now()
),
(
  'c0ffee00-0000-0000-0000-000000000003',
  'aaaa0002-0000-0000-0000-000000000002',
  '66666666-6666-6666-6666-666666666666',
  now()
),
(
  'c0ffee00-0000-0000-0000-000000000005',
  'aaaa0004-0000-0000-0000-000000000004',
  '66666666-6666-6666-6666-666666666666',
  now()
);

INSERT INTO public.request_target_dzos (
  id, request_id, dzo_id, created_at
) VALUES
(
  'c0ffee00-0000-0000-0000-000000000002',
  'aaaa0001-0000-0000-0000-000000000001',
  '33333333-3333-3333-3333-333333333333',
  now()
),
(
  'c0ffee00-0000-0000-0000-000000000004',
  'aaaa0002-0000-0000-0000-000000000002',
  '33333333-3333-3333-3333-333333333333',
  now()
),
(
  'c0ffee00-0000-0000-0000-000000000006',
  'aaaa0004-0000-0000-0000-000000000004',
  '33333333-3333-3333-3333-333333333333',
  now()
);
