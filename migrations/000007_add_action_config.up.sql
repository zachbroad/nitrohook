ALTER TABLE actions ADD COLUMN config JSONB;
ALTER TABLE actions DROP CONSTRAINT chk_action_type;
ALTER TABLE actions ADD CONSTRAINT chk_action_type CHECK (type IN ('webhook', 'javascript', 'slack', 'smtp', 'twilio'));
