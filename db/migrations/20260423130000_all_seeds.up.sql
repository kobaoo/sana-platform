---------------------------------------------------
-- CLIENT
---------------------------------------------------
INSERT INTO public.clients (
  id, name, domain, language, user_limit, is_active, created_at
) VALUES (
  '11111111-1111-1111-1111-111111111111',
  'Demo Client',
  'demo.local',
  'ru',
  500,
  true,
  now()
);

---------------------------------------------------
-- ORGANIZATION
---------------------------------------------------
INSERT INTO public.organizations (
  id, name, code, type, is_active, created_at, updated_at, parent_id
) VALUES (
  '22222222-2222-2222-2222-222222222222',
  'Main Organization',
  'MAIN_ORG',
  'subsidiary',
  true,
  now(),
  now(),
  NULL
);

---------------------------------------------------
-- DZO
---------------------------------------------------
INSERT INTO public.dzo_organizations (
  id, client_id, name, short_name, bin, is_active, created_at, updated_at
) VALUES (
  '33333333-3333-3333-3333-333333333333',
  '11111111-1111-1111-1111-111111111111',
  'Demo DZO',
  'DDZO',
  '123456789012',
  true,
  now(),
  now()
);

---------------------------------------------------
-- CATEGORY
---------------------------------------------------
INSERT INTO public.categories (
  id, name, description
) VALUES (
  '44444444-4444-4444-4444-444444444444',
  'Safety Training',
  'Basic training'
);

---------------------------------------------------
-- USER
---------------------------------------------------
INSERT INTO public.users (
  id, keycloak_user_id, email, role, dzo_id,
  is_active, is_onboarded, created_at, updated_at, client_id
) VALUES (
  '55555555-5555-5555-5555-555555555555',
  'kc-demo-user',
  'admin@demo.local',
  'ADM',
  '33333333-3333-3333-3333-333333333333',
  true,
  true,
  now(),
  now(),
  '11111111-1111-1111-1111-111111111111'
);

-- HR user for DZO
INSERT INTO public.users (
  id, keycloak_user_id, email, role, dzo_id,
  is_active, is_onboarded, created_at, updated_at, client_id
) VALUES (
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  'kc-demo-hr',
  'hr@demo.local',
  'HR',
  '33333333-3333-3333-3333-333333333333',
  true,
  true,
  now(),
  now(),
  '11111111-1111-1111-1111-111111111111'
);

---------------------------------------------------
-- EMPLOYEE
---------------------------------------------------
INSERT INTO public.employees (
  id, client_id, position, full_name, short_name,
  department, direction, email, internal_phone,
  birth_date, is_active, user_id, dzo_id, is_deleted
) VALUES (
  '66666666-6666-6666-6666-666666666666',
  '11111111-1111-1111-1111-111111111111',
  'Engineer',
  'Ivan Ivanov',
  'Ivanov I.',
  'Production',
  'Technical',
  'ivanov@demo.local',
  '101',
  '1998-05-12',
  true,
  '55555555-5555-5555-5555-555555555555',
  '33333333-3333-3333-3333-333333333333',
  false
);

---------------------------------------------------
-- SUPPLIER
---------------------------------------------------
INSERT INTO public.suppliers (
  id, client_id, type, name, bin_or_iin, local_content_pct, is_active
) VALUES (
  '77777777-7777-7777-7777-777777777777',
  '11111111-1111-1111-1111-111111111111',
  'LEGAL',
  'Demo Supplier',
  '987654321012',
  75.50,
  true
);

---------------------------------------------------
-- CONTRACT
---------------------------------------------------
INSERT INTO public.contract_suppliers (
  id, supplier_id, contract_number, vat_flag,
  signed_date, amount, amount_currency, currency,
  balance_at_year_end, amendment_number, amendment_date,
  amendment_amount, total_with_amendment, remaining_amount,
  is_active, created_at, updated_at,
  file_key, file_name, file_size, file_mime_type, end_date
) VALUES (
  '88888888-8888-8888-8888-888888888888',
  '77777777-7777-7777-7777-777777777777',
  'CTR-001',
  true,
  '2026-01-15',
  1000000,
  1000000,
  'KZT',
  400000,
  'AM-01',
  '2026-02-10',
  200000,
  1200000,
  600000,
  true,
  now(),
  now(),
  'contracts/demo.pdf',
  'demo.pdf',
  123456,
  'application/pdf',
  '2026-12-31'
);

---------------------------------------------------
-- EXTERNAL TRAINING
---------------------------------------------------
INSERT INTO public.external_training_events (
  id, name, format, capacity,
  supplier_cost_vat, start_date,
  is_active, created_at,
  category_id, contract_id, supplier_id, responsible_user_id
) VALUES (
  '99999999-9999-9999-9999-999999999999',
  'Safety Training External',
  'OFFLINE',
  25,
  350000,
  now() + interval '5 days',
  true,
  now(),
  '44444444-4444-4444-4444-444444444444',
  '88888888-8888-8888-8888-888888888888',
  '77777777-7777-7777-7777-777777777777',
  '55555555-5555-5555-5555-555555555555'
);

---------------------------------------------------
-- TRAINING EVENT
---------------------------------------------------
INSERT INTO public.training_events (
  id, title, start_date, end_date,
  location_type, location_city,
  category_id, direction, dzo_id,
  participants_count, cost_per_person_vat,
  supplier_id, supplier_contract_id
) VALUES (
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  'Internal Training',
  now(),
  now() + interval '2 days',
  'OFFLINE',
  'Astana',
  '44444444-4444-4444-4444-444444444444',
  'Management',
  '33333333-3333-3333-3333-333333333333',
  20,
  50000,
  '77777777-7777-7777-7777-777777777777',
  '88888888-8888-8888-8888-888888888888'
);

---------------------------------------------------
-- PARTICIPANT
---------------------------------------------------
INSERT INTO public.training_participants (
  id, event_id, employee_id, status
) VALUES (
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  '66666666-6666-6666-6666-666666666666',
  'ENROLLED'
);

---------------------------------------------------
-- REQUEST
---------------------------------------------------
INSERT INTO public.requests (
  id,
  initiator_id,
  entity_id,
  entity_type,
  request_type,
  title,
  step,
  created_at,
  updated_at,
  status
) VALUES (
  'cccccccc-cccc-cccc-cccc-cccccccccccc',
  '55555555-5555-5555-5555-555555555555',
  '99999999-9999-9999-9999-999999999999',
  'EXTERNAL_TRAINING',
  'MAIN',
  'Demo External Training Request',
  0,
  now(),
  now(),
  'PENDING'
);

---------------------------------------------------
-- REJECTED REQUEST (для тестирования RecreateRejectedRequest)
---------------------------------------------------
INSERT INTO public.requests (
  id,
  initiator_id,
  entity_id,
  entity_type,
  request_type,
  kind,
  title,
  category,
  format,
  responsible_admin_id,
  training_date,
  cost_amount,
  cost_mode,
  step,
  status,
  created_at,
  updated_at
) VALUES (
  'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
  '55555555-5555-5555-5555-555555555555',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  'TRAINING_EVENT',
  'MAIN',
  'REGULAR',
  'Rejected Training Request',
  'Safety Training',
  'OFFLINE',
  '55555555-5555-5555-5555-555555555555',
  now() + interval '10 days',
  100000,
  'PER_EMPLOYEE',
  0,
  'REJECTED',
  now() - interval '2 days',
  now() - interval '1 day'
);

---------------------------------------------------
-- REQUEST PARTICIPANTS (для отклоненной заявки)
---------------------------------------------------
INSERT INTO public.request_participants (
  id, request_id, employee_id, created_at
) VALUES (
  'ffffffff-ffff-ffff-ffff-ffffffffffff',
  'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
  '66666666-6666-6666-6666-666666666666',
  now()
);

---------------------------------------------------
-- REQUEST TARGET DZO (для отклоненной заявки)
---------------------------------------------------
INSERT INTO public.request_target_dzos (
  id, request_id, dzo_id, created_at
) VALUES (
  '10101010-1010-1010-1010-101010101010',
  'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
  '33333333-3333-3333-3333-333333333333',
  now()
);

---------------------------------------------------
-- HISTORY
---------------------------------------------------
INSERT INTO public.contract_supplier_histories (
  history_id, contract_id, operation_type,
  changed_at, changed_by, snapshot, diff
) VALUES (
  'dddddddd-dddd-dddd-dddd-dddddddddddd',
  '88888888-8888-8888-8888-888888888888',
  'CREATE',
  now(),
  '55555555-5555-5555-5555-555555555555',
  '{"contract":"CTR-001"}',
  '{"action":"created"}'
);