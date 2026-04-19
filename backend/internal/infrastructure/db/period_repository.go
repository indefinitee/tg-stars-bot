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

// PeriodRepository implements domain.PeriodRepository using PostgreSQL
type PeriodRepository struct {
	pool *pgxpool.Pool
}

// NewPeriodRepository creates a new PeriodRepository
func NewPeriodRepository(pool *pgxpool.Pool) *PeriodRepository {
	return &PeriodRepository{pool: pool}
}

// GetByID retrieves a period by its UUID
func (r *PeriodRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Period, error) {
	var period domain.Period
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, start_date, end_date, is_active, votes_per_employee, 
		       vote_weight, created_at, updated_at
		FROM periods WHERE id = $1
	`, id).Scan(
		&period.ID, &period.Name, &period.StartDate, &period.EndDate,
		&period.IsActive, &period.VotesPerEmployee, &period.VoteWeight,
		&period.CreatedAt, &period.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &period, nil
}

// GetActive retrieves the currently active period
func (r *PeriodRepository) GetActive(ctx context.Context) (*domain.Period, error) {
	var period domain.Period
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, start_date, end_date, is_active, votes_per_employee, 
		       vote_weight, created_at, updated_at
		FROM periods WHERE is_active = TRUE LIMIT 1
	`).Scan(
		&period.ID, &period.Name, &period.StartDate, &period.EndDate,
		&period.IsActive, &period.VotesPerEmployee, &period.VoteWeight,
		&period.CreatedAt, &period.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &period, nil
}

// List retrieves all periods
func (r *PeriodRepository) List(ctx context.Context) ([]*domain.Period, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, start_date, end_date, is_active, votes_per_employee, 
		       vote_weight, created_at, updated_at
		FROM periods ORDER BY start_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []*domain.Period
	for rows.Next() {
		var period domain.Period
		err := rows.Scan(
			&period.ID, &period.Name, &period.StartDate, &period.EndDate,
			&period.IsActive, &period.VotesPerEmployee, &period.VoteWeight,
			&period.CreatedAt, &period.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		periods = append(periods, &period)
	}
	return periods, rows.Err()
}

// Create creates a new period
func (r *PeriodRepository) Create(ctx context.Context, period *domain.Period) error {
	period.ID = uuid.New()
	period.CreatedAt = time.Now()
	period.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO periods (id, name, start_date, end_date, is_active, votes_per_employee, 
		                     vote_weight, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, period.ID, period.Name, period.StartDate, period.EndDate,
		period.IsActive, period.VotesPerEmployee, period.VoteWeight,
		period.CreatedAt, period.UpdatedAt)
	return err
}

// Update updates an existing period
func (r *PeriodRepository) Update(ctx context.Context, period *domain.Period) error {
	period.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE periods SET
			name = $2,
			start_date = $3,
			end_date = $4,
			is_active = $5,
			votes_per_employee = $6,
			vote_weight = $7,
			updated_at = $8
		WHERE id = $1
	`, period.ID, period.Name, period.StartDate, period.EndDate,
		period.IsActive, period.VotesPerEmployee, period.VoteWeight, period.UpdatedAt)
	return err
}

// SetActive sets the period as active (deactivates all others)
func (r *PeriodRepository) SetActive(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE periods SET is_active = FALSE WHERE is_active = TRUE`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE periods SET is_active = TRUE WHERE id = $1`, id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Close closes a period (makes it inactive)
func (r *PeriodRepository) Close(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE periods SET is_active = FALSE, updated_at = NOW() WHERE id = $1
	`, id)
	return err
}
