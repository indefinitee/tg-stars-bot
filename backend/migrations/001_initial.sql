-- Schema for Stars Recognition System
-- Roles: employee, hr, manager

-- Enum for user roles
CREATE TYPE user_role AS ENUM ('employee', 'hr', 'manager');

-- Users table (synced from Bitrix24)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bitrix_id INTEGER UNIQUE NOT NULL,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255) NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    email VARCHAR(255),
    role user_role NOT NULL DEFAULT 'employee',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_voting_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_bitrix_id ON users(bitrix_id);
CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_is_active ON users(is_active);

-- Periods table (voting periods, every 2 months)
CREATE TABLE periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    votes_per_employee INTEGER NOT NULL DEFAULT 3,
    vote_weight INTEGER NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_period_dates EXCLUDE USING gist (
        daterange(start_date, end_date, '[]') WITH =
    ) WHERE (is_active = TRUE)
);

CREATE INDEX idx_periods_active ON periods(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_periods_dates ON periods(start_date, end_date);

-- Votes table (stars given by employees)
CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id),
    receiver_id UUID NOT NULL REFERENCES users(id),
    period_id UUID NOT NULL REFERENCES periods(id),
    weight INTEGER NOT NULL DEFAULT 5,
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_sender_receiver_period UNIQUE (sender_id, receiver_id, period_id),
    CONSTRAINT cannot_vote_for_self CHECK (sender_id != receiver_id),
    CONSTRAINT valid_weight CHECK (weight > 0 AND weight <= 10)
);

CREATE INDEX idx_votes_sender ON votes(sender_id);
CREATE INDEX idx_votes_receiver ON votes(receiver_id);
CREATE INDEX idx_votes_period ON votes(period_id);
CREATE INDEX idx_votes_sender_period ON votes(sender_id, period_id);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_periods_updated_at
    BEFORE UPDATE ON periods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert a default active period for testing
INSERT INTO periods (name, start_date, end_date, is_active, votes_per_employee, vote_weight)
VALUES ('Q1-Q2 2026', '2026-01-01', '2026-02-28', TRUE, 3, 5);
