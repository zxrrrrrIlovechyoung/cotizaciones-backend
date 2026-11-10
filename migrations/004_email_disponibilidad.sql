CREATE INDEX IF NOT EXISTS idx_users_email_normalizado ON users (lower(btrim(email)));
CREATE OR REPLACE VIEW registro_emails AS
SELECT lower(btrim(email)) AS email FROM users;
