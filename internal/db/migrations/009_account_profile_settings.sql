ALTER TABLE users ADD COLUMN gender TEXT NOT NULL DEFAULT 'secret';
ALTER TABLE users ADD COLUMN phone_number TEXT;
ALTER TABLE users ADD COLUMN phone_number_normalized TEXT;
ALTER TABLE users ADD COLUMN email_public INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN phone_number_public INTEGER NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_number_normalized
ON users(phone_number_normalized)
WHERE phone_number_normalized IS NOT NULL AND phone_number_normalized <> '';
