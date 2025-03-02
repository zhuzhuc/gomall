-- Add two-factor authentication columns
ALTER TABLE users
ADD COLUMN two_factor_enabled BOOLEAN DEFAULT FALSE,
ADD COLUMN two_factor_secret VARCHAR(32),
ADD COLUMN two_factor_backup_codes TEXT[];

-- Add index for performance
CREATE INDEX idx_users_2fa_enabled ON users(two_factor_enabled) WHERE two_factor_enabled = TRUE;