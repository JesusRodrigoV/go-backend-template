CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- Table: users
-- ==========================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    pwd VARCHAR(255),
    verified BOOLEAN DEFAULT FALSE,
    vers INT DEFAULT 1,
    mfa BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_email ON users(LOWER(email));

CREATE TRIGGER trg_users_updated
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_modified_column();

-- ==========================================
-- Table: socials
-- ==========================================
CREATE TABLE socials (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    provider VARCHAR(50) NOT NULL,
    prov_id VARCHAR(100) NOT NULL,
    raw JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT fk_socials_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE,

    CONSTRAINT uq_provider_id UNIQUE (provider, prov_id)
);

-- ==========================================
-- Table: sessions
-- ==========================================
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    hash VARCHAR(255) NOT NULL,
    device VARCHAR(255),
    ip INET,
    exp TIMESTAMP NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_exp ON sessions(exp);

-- ==========================================
-- Table: otp
-- ==========================================
CREATE TABLE otp (
    user_id UUID PRIMARY KEY,
    secret VARCHAR(255) NOT NULL,
    backup TEXT[],
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT fk_otp_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
);

CREATE TRIGGER trg_otp_updated
BEFORE UPDATE ON otp
FOR EACH ROW EXECUTE FUNCTION update_modified_column();

-- ==========================================
-- Table: logs
-- ==========================================
CREATE TABLE logs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID,
    event VARCHAR(50) NOT NULL,
    meta JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT fk_logs_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX idx_logs_created ON logs(created_at DESC);
