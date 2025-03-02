-- Remove two-factor authentication columns
DROP INDEX IF EXISTS idx_users_2fa_enabled;

ALTER TABLE users
DROP COLUMN IF EXISTS two_factor_backup_codes,
DROP COLUMN IF EXISTS two_factor_secret,
DROP COLUMN IF EXISTS two_factor_enabled;