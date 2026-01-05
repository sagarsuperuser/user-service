-- users table:
CREATE TABLE users (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  email         VARCHAR(320) NOT NULL,
  email_locked  BOOLEAN NOT NULL, -- TRUE for Google auth users
  status        ENUM('active','disabled') NOT NULL DEFAULT 'active',
  role          ENUM('user','admin') NOT NULL DEFAULT 'user',
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_users_email (email)
);


-- auth_identities: supports local password OR google sso
CREATE TABLE auth_identities (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id           BIGINT UNSIGNED NOT NULL,
  provider          ENUM('local','google') NOT NULL,
  provider_subject  VARCHAR(255) NULL,  -- for Google: "sub"
  password_hash     VARCHAR(255) NULL,  -- for password: bcrypt hash
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),

  CONSTRAINT fk_auth_identities_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE,

  UNIQUE KEY uq_user_provider (user_id, provider),
  UNIQUE KEY uq_provider_subject (provider, provider_subject)
);


-- user_profiles table: stores additional user info
CREATE TABLE user_profiles (
  user_id      BIGINT UNSIGNED NOT NULL,
  full_name    VARCHAR(255) NOT NULL DEFAULT '',
  telephone    VARCHAR(30)  NOT NULL DEFAULT '',
  avatar_url   VARCHAR(2048) NOT NULL DEFAULT '',
  created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id),
  CONSTRAINT fk_profiles_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
);

-- sessions table: stores user sessions
CREATE TABLE sessions (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id     BIGINT UNSIGNED NOT NULL,
  token_hash  BINARY(32) NOT NULL,          -- sha256(token)
  expires_at  TIMESTAMP NOT NULL,
  revoked_at  TIMESTAMP NULL,
  created_at  TIMESTAMP NOT NULL,
  PRIMARY KEY (id),

  CONSTRAINT fk_sessions_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE,

  UNIQUE KEY uq_sessions_token_hash (token_hash),
  KEY idx_sessions_expires (expires_at)
);