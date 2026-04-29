-- Rollback for seed_requests_participants.up.sql

DELETE FROM public.request_participants
WHERE id IN (
  'c0ffee00-0000-0000-0000-000000000001',
  'c0ffee00-0000-0000-0000-000000000003',
  'c0ffee00-0000-0000-0000-000000000005'
);

DELETE FROM public.request_target_dzos
WHERE id IN (
  'c0ffee00-0000-0000-0000-000000000002',
  'c0ffee00-0000-0000-0000-000000000004',
  'c0ffee00-0000-0000-0000-000000000006'
);
