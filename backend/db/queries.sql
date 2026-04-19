-- User queries
CREATE OR REPLACE FUNCTION get_user_by_id(user_id UUID)
RETURNS users AS $$
SELECT * FROM users WHERE id = user_id;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION get_user_by_telegram_id(tg_id BIGINT)
RETURNS users AS $$
SELECT * FROM users WHERE telegram_id = tg_id;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION get_user_by_bitrix_id(bx_id INTEGER)
RETURNS users AS $$
SELECT * FROM users WHERE bitrix_id = bx_id;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION list_users()
RETURNS SETOF users AS $$
SELECT * FROM users WHERE is_active = TRUE ORDER BY first_name, last_name;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION list_users_by_role(user_role user_role)
RETURNS SETOF users AS $$
SELECT * FROM users WHERE role = user_role AND is_active = TRUE ORDER BY first_name, last_name;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION create_user(
    p_bitrix_id INTEGER,
    p_telegram_id BIGINT,
    p_username VARCHAR,
    p_first_name VARCHAR,
    p_last_name VARCHAR,
    p_email VARCHAR,
    p_role user_role
)
RETURNS users AS $$
INSERT INTO users (bitrix_id, telegram_id, username, first_name, last_name, email, role)
VALUES (p_bitrix_id, p_telegram_id, p_username, p_first_name, p_last_name, p_email, p_role)
RETURNING *;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION update_user(p_user users)
RETURNS users AS $$
UPDATE users SET
    username = p_user.username,
    first_name = p_user.first_name,
    last_name = p_user.last_name,
    email = p_user.email,
    role = p_user.role,
    is_active = p_user.is_active,
    is_voting_active = p_user.is_voting_active
WHERE id = p_user.id
RETURNING *;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION upsert_from_bitrix(
    p_bitrix_id INTEGER,
    p_telegram_id BIGINT,
    p_username VARCHAR,
    p_first_name VARCHAR,
    p_last_name VARCHAR,
    p_email VARCHAR
)
RETURNS users AS $$
INSERT INTO users (bitrix_id, telegram_id, username, first_name, last_name, email)
VALUES (p_bitrix_id, p_telegram_id, p_username, p_first_name, p_last_name, p_email)
ON CONFLICT (bitrix_id) DO UPDATE SET
    telegram_id = EXCLUDED.telegram_id,
    username = EXCLUDED.username,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    email = EXCLUDED.email,
    updated_at = NOW()
RETURNING *;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION set_user_voting_active(user_id UUID, active BOOLEAN)
RETURNS VOID AS $$
UPDATE users SET is_voting_active = active WHERE id = user_id;
$$ LANGUAGE SQL;

-- Period queries
CREATE OR REPLACE FUNCTION get_period_by_id(period_id UUID)
RETURNS periods AS $$
SELECT * FROM periods WHERE id = period_id;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION get_active_period()
RETURNS periods AS $$
SELECT * FROM periods WHERE is_active = TRUE LIMIT 1;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION list_periods()
RETURNS SETOF periods AS $$
SELECT * FROM periods ORDER BY start_date DESC;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION create_period(
    p_name VARCHAR,
    p_start_date DATE,
    p_end_date DATE,
    p_votes_per_employee INTEGER,
    p_vote_weight INTEGER
)
RETURNS periods AS $$
INSERT INTO periods (name, start_date, end_date, votes_per_employee, vote_weight)
VALUES (p_name, p_start_date, p_end_date, p_votes_per_employee, p_vote_weight)
RETURNING *;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION update_period(p_period periods)
RETURNS periods AS $$
UPDATE periods SET
    name = p_period.name,
    start_date = p_period.start_date,
    end_date = p_period.end_date,
    votes_per_employee = p_period.votes_per_employee,
    vote_weight = p_period.vote_weight
WHERE id = p_period.id
RETURNING *;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION set_period_active(period_id UUID)
RETURNS VOID AS $$
UPDATE periods SET is_active = FALSE WHERE is_active = TRUE;
UPDATE periods SET is_active = TRUE WHERE id = period_id;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION close_period(period_id UUID)
RETURNS VOID AS $$
UPDATE periods SET is_active = FALSE WHERE id = period_id;
$$ LANGUAGE SQL;

-- Vote queries
CREATE OR REPLACE FUNCTION create_vote(
    p_sender_id UUID,
    p_receiver_id UUID,
    p_period_id UUID,
    p_weight INTEGER,
    p_message TEXT
)
RETURNS votes AS $$
INSERT INTO votes (sender_id, receiver_id, period_id, weight, message)
VALUES (p_sender_id, p_receiver_id, p_period_id, p_weight, p_message)
RETURNING *;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION get_vote_by_id(vote_id UUID)
RETURNS votes AS $$
SELECT * FROM votes WHERE id = vote_id;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION list_votes_by_period(period_id UUID)
RETURNS SETOF votes AS $$
SELECT * FROM votes WHERE period_id = period_id ORDER BY created_at DESC;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION list_votes_by_sender(sender_id UUID, p_period_id UUID)
RETURNS SETOF votes AS $$
SELECT * FROM votes WHERE sender_id = sender_id AND period_id = p_period_id ORDER BY created_at DESC;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION list_votes_by_receiver(receiver_id UUID, p_period_id UUID)
RETURNS TABLE (
    id UUID,
    sender_id UUID,
    receiver_id UUID,
    period_id UUID,
    weight INTEGER,
    message TEXT,
    created_at TIMESTAMPTZ,
    sender_name VARCHAR
) AS $$
SELECT v.id, v.sender_id, v.receiver_id, v.period_id, v.weight, v.message, v.created_at,
       CONCAT(u.first_name, ' ', COALESCE(u.last_name, '')) as sender_name
FROM votes v
JOIN users u ON v.sender_id = u.id
WHERE v.receiver_id = receiver_id AND v.period_id = p_period_id
ORDER BY v.created_at DESC;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION count_votes_by_sender(sender_id UUID, p_period_id UUID)
RETURNS INTEGER AS $$
SELECT COUNT(*) FROM votes WHERE sender_id = sender_id AND period_id = p_period_id;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION has_voted_for(sender_id UUID, receiver_id UUID, p_period_id UUID)
RETURNS BOOLEAN AS $$
SELECT EXISTS(SELECT 1 FROM votes WHERE sender_id = sender_id AND receiver_id = receiver_id AND period_id = p_period_id);
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION get_user_stats(p_user_id UUID, p_period_id UUID)
RETURNS TABLE (
    user_id UUID,
    user_name VARCHAR,
    total_votes INTEGER,
    total_weight BIGINT
) AS $$
SELECT 
    u.id as user_id,
    CONCAT(u.first_name, ' ', COALESCE(u.last_name, '')) as user_name,
    COUNT(v.id) as total_votes,
    COALESCE(SUM(v.weight), 0) as total_weight
FROM users u
LEFT JOIN votes v ON u.id = v.receiver_id AND v.period_id = p_period_id
WHERE u.id = p_user_id
GROUP BY u.id, u.first_name, u.last_name;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION get_period_leaderboard(p_period_id UUID)
RETURNS TABLE (
    user_id UUID,
    user_name VARCHAR,
    total_votes INTEGER,
    total_weight BIGINT
) AS $$
SELECT 
    u.id as user_id,
    CONCAT(u.first_name, ' ', COALESCE(u.last_name, '')) as user_name,
    COUNT(v.id) as total_votes,
    COALESCE(SUM(v.weight), 0) as total_weight
FROM users u
LEFT JOIN votes v ON u.id = v.receiver_id AND v.period_id = p_period_id
WHERE u.is_active = TRUE
GROUP BY u.id, u.first_name, u.last_name
ORDER BY total_weight DESC, total_votes DESC;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION get_votes_received(p_user_id UUID, p_period_id UUID)
RETURNS TABLE (
    vote_id UUID,
    sender_id UUID,
    receiver_id UUID,
    period_id UUID,
    weight INTEGER,
    message TEXT,
    created_at TIMESTAMPTZ,
    sender_name VARCHAR
) AS $$
SELECT v.id, v.sender_id, v.receiver_id, v.period_id, v.weight, v.message, v.created_at,
       CONCAT(u.first_name, ' ', COALESCE(u.last_name, '')) as sender_name
FROM votes v
JOIN users u ON v.sender_id = u.id
WHERE v.receiver_id = p_user_id AND v.period_id = p_period_id
ORDER BY v.created_at DESC;
$$ LANGUAGE SQL;
