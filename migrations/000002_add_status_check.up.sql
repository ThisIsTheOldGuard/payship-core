ALTER TABLE orders 
ADD CONSTRAINT chk_order_status 
CHECK (status IN ('pending', 'processing', 'completed', 'cancelled'));