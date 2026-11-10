-- Enforce the new limit on writes without truncating existing profiles.
ALTER TABLE perfiles_cuenta ADD CONSTRAINT perfiles_cuenta_comercial_max_50
    CHECK (char_length(btrim(comercial)) <= 50) NOT VALID;
