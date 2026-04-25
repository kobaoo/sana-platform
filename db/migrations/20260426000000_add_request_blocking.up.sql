-- Add fields for request blocking after recreation
ALTER TABLE requests 
ADD COLUMN replaced_by_request_id UUID,
ADD COLUMN is_blocked BOOLEAN NOT NULL DEFAULT false;

-- Add foreign key constraint
ALTER TABLE requests
ADD CONSTRAINT fk_requests_replaced_by
FOREIGN KEY (replaced_by_request_id) 
REFERENCES requests(id) 
ON DELETE SET NULL;

-- Add index for performance
CREATE INDEX idx_requests_replaced_by ON requests(replaced_by_request_id);
CREATE INDEX idx_requests_is_blocked ON requests(is_blocked);

-- Add comment
COMMENT ON COLUMN requests.replaced_by_request_id IS 'ID новой заявки, которая заменила эту отклоненную';
COMMENT ON COLUMN requests.is_blocked IS 'Заблокирована ли заявка после пересоздания';
