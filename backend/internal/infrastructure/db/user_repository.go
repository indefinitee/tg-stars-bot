package db

import (
	"context"
	"errors"
	"time"

	"tg-stars-bot/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implements domain.UserRepository using PostgreSQL
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// GetByID retrieves a user by their UUID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, bitrix_id, telegram_id, username, first_name, last_name, 
		       email, role, is_active, is_voting_active, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.BitrixID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.Email, &user.Role, &user.IsActive, &user.IsVotingActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByTelegramID retrieves a user by their Telegram ID
func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, bitrix_id, telegram_id, username, first_name, last_name, 
		       email, role, is_active, is_voting_active, created_at, updated_at
		FROM users WHERE telegram_id = $1
	`, telegramID).Scan(
		&user.ID, &user.BitrixID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.Email, &user.Role, &user.IsActive, &user.IsVotingActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByBitrixID retrieves a user by their Bitrix24 ID
func (r *UserRepository) GetByBitrixID(ctx context.Context, bitrixID int) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, bitrix_id, telegram_id, username, first_name, last_name, 
		       email, role, is_active, is_voting_active, created_at, updated_at
		FROM users WHERE bitrix_id = $1
	`, bitrixID).Scan(
		&user.ID, &user.BitrixID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.Email, &user.Role, &user.IsActive, &user.IsVotingActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List retrieves all active users
func (r *UserRepository) List(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, bitrix_id, telegram_id, username, first_name, last_name, 
		       email, role, is_active, is_voting_active, created_at, updated_at
		FROM users WHERE is_active = TRUE ORDER BY first_name, last_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID, &user.BitrixID, &user.TelegramID, &user.Username, &user.FirstName,
			&user.LastName, &user.Email, &user.Role, &user.IsActive, &user.IsVotingActive,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

// ListByRole retrieves users by role
func (r *UserRepository) ListByRole(ctx context.Context, role domain.UserRole) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, bitrix_id, telegram_id, username, first_name, last_name, 
		       email, role, is_active, is_voting_active, created_at, updated_at
		FROM users WHERE role = $1 AND is_active = TRUE ORDER BY first_name, last_name
	`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID, &user.BitrixID, &user.TelegramID, &user.Username, &user.FirstName,
			&user.LastName, &user.Email, &user.Role, &user.IsActive, &user.IsVotingActive,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, bitrix_id, telegram_id, username, first_name, last_name, 
		                   email, role, is_active, is_voting_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, user.ID, user.BitrixID, user.TelegramID, user.Username, user.FirstName,
		user.LastName, user.Email, user.Role, user.IsActive, user.IsVotingActive,
		user.CreatedAt, user.UpdatedAt)
	return err
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET
			username = $2,
			first_name = $3,
			last_name = $4,
			email = $5,
			role = $6,
			is_active = $7,
			is_voting_active = $8,
			updated_at = $9
		WHERE id = $1
	`, user.ID, user.Username, user.FirstName, user.LastName, user.Email,
		user.Role, user.IsActive, user.IsVotingActive, user.UpdatedAt)
	return err
}

// UpsertFromBitrix creates or updates a user from Bitrix24 data
func (r *UserRepository) UpsertFromBitrix(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (bitrix_id, telegram_id, username, first_name, last_name, email, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (bitrix_id) DO UPDATE SET
			telegram_id = EXCLUDED.telegram_id,
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			email = EXCLUDED.email,
			updated_at = EXCLUDED.updated_at
	`, user.BitrixID, user.TelegramID, user.Username, user.FirstName,
		user.LastName, user.Email, user.UpdatedAt)
	return err
}

// SetVotingActive sets whether a user can vote
func (r *UserRepository) SetVotingActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET is_voting_active = $2, updated_at = NOW() WHERE id = $1
	`, id, active)
	return err
}
