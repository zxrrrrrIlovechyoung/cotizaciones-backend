CREATE TABLE perfiles_cuenta (
 user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 tipo varchar(7) NOT NULL CHECK (tipo IN ('Empresa','Persona')),
 comercial varchar(100) NOT NULL CHECK (char_length(btrim(comercial)) BETWEEN 2 AND 100),
 razon_social varchar(150) NOT NULL DEFAULT '',
 rfc varchar(13) NOT NULL DEFAULT '',
 telefono varchar(10) NOT NULL DEFAULT '' CHECK (telefono = '' OR telefono ~ '^[0-9]{10}$'),
 direccion varchar(250) NOT NULL DEFAULT '',
 CHECK ((tipo='Empresa' AND char_length(btrim(razon_social)) BETWEEN 2 AND 150) OR (tipo='Persona' AND razon_social='')),
 CHECK (rfc='' OR (tipo='Empresa' AND rfc ~ '^[A-ZÑ&]{3}[0-9]{6}[A-Z0-9]{3}$') OR (tipo='Persona' AND rfc ~ '^[A-ZÑ&]{4}[0-9]{6}[A-Z0-9]{3}$'))
);
