CREATE TABLE request_budget_transactions (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contract_suppliers(id),
    amount DOUBLE PRECISION NOT NULL,
    operation_type VARCHAR(50) NOT NULL,
    created_by UUID REFERENCES users(id),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX request_budget_transactions_request_id_idx
    ON request_budget_transactions(request_id);

CREATE INDEX request_budget_transactions_operation_type_idx
    ON request_budget_transactions(operation_type);