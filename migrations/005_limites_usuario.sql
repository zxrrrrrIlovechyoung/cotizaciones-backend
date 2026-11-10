-- Abort if existing values exceed these limits; never truncate user data.
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM users WHERE char_length(nombre) > 60 OR char_length(email) > 100) THEN
    RAISE EXCEPTION 'Hay usuarios que exceden los nuevos limites';
  END IF;
END $$;
DROP VIEW registro_emails;
ALTER TABLE users ALTER COLUMN nombre TYPE varchar(60);
ALTER TABLE users ALTER COLUMN email TYPE varchar(100);
CREATE VIEW registro_emails AS SELECT lower(btrim(email)) AS email FROM users;
-- password_hash remains TEXT: bcrypt hashes are longer than passwords.
