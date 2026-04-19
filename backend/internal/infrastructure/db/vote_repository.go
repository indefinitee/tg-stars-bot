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

// VoteRepository implements domain.VoteRepository using PostgreSQL
type VoteRepository struct {
	pool *pgxpool.Pool
}

// NewVoteRepository creates a new VoteRepository
func NewVoteRepository(pool *pgxpool.Pool) *VoteRepository {
	return &VoteRepository{pool: pool}
}

// Create creates a new vote
func (r *VoteRepository) Create(ctx context.Context, vote *domain.Vote) error {
	vote.ID = uuid.New()
	vote.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO votes (id, sender_id, receiver_id, period_id, weight, message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, vote.ID, vote.SenderID, vote.ReceiverID, vote.PeriodID,
		vote.Weight, vote.Message, vote.CreatedAt)
	return err
}

// GetByID retrieves a vote by its UUID
func (r *VoteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vote, error) {
	var vote domain.Vote
	err := r.pool.QueryRow(ctx, `
		SELECT id, sender_id, receiver_id, period_id, weight, message, created_at
		FROM votes WHERE id = $1
	`, id).Scan(
		&vote.ID, &vote.SenderID, &vote.ReceiverID, &vote.PeriodID,
		&vote.Weight, &vote.Message, &vote.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &vote, nil
}

// ListByPeriod retrieves all votes for a period
func (r *VoteRepository) ListByPeriod(ctx context.Context, periodID uuid.UUID) ([]*domain.Vote, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sender_id, receiver_id, period_id, weight, message, created_at
		FROM votes WHERE period_id = $1 ORDER BY created_at DESC
	`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*domain.Vote
	for rows.Next() {
		var vote domain.Vote
		err := rows.Scan(
			&vote.ID, &vote.SenderID, &vote.ReceiverID, &vote.PeriodID,
			&vote.Weight, &vote.Message, &vote.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		votes = append(votes, &vote)
	}
	return votes, rows.Err()
}

// ListBySender retrieves all votes sent by a user in a period
func (r *VoteRepository) ListBySender(ctx context.Context, senderID, periodID uuid.UUID) ([]*domain.Vote, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sender_id, receiver_id, period_id, weight, message, created_at
		FROM votes WHERE sender_id = $1 AND period_id = $2 ORDER BY created_at DESC
	`, senderID, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*domain.Vote
	for rows.Next() {
		var vote domain.Vote
		err := rows.Scan(
			&vote.ID, &vote.SenderID, &vote.ReceiverID, &vote.PeriodID,
			&vote.Weight, &vote.Message, &vote.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		votes = append(votes, &vote)
	}
	return votes, rows.Err()
}

// ListByReceiver retrieves all votes received by a user in a period
func (r *VoteRepository) ListByReceiver(ctx context.Context, receiverID, periodID uuid.UUID) ([]*domain.VoteWithUsers, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.sender_id, v.receiver_id, v.period_id, v.weight, v.message, v.created_at,
		       CONCAT(u.first_name, ' ', COALESCE(u.last_name, '')) as sender_name
		FROM votes v
		JOIN users u ON v.sender_id = u.id
		WHERE v.receiver_id = $1 AND v.period_id = $2
		ORDER BY v.created_at DESC
	`, receiverID, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*domain.VoteWithUsers
	for rows.Next() {
		var vote domain.VoteWithUsers
		err := rows.Scan(
			&vote.ID, &vote.SenderID, &vote.ReceiverID, &vote.PeriodID,
			&vote.Weight, &vote.Message, &vote.CreatedAt, &vote.SenderName,
		)
		if err != nil {
			return nil, err
		}
		votes = append(votes, &vote)
	}
	return votes, rows.Err()
}

// CountBySender counts votes sent by a user in a period
func (r *VoteRepository) CountBySender(ctx context.Context, senderID, periodID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM votes WHERE sender_id = $1 AND period_id = $2
	`, senderID, periodID).Scan(&count)
	return count, err
}

// HasVotedFor checks if a user has already voted for another user in a period
func (r *VoteRepository) HasVotedFor(ctx context.Context, senderID, receiverID, periodID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM votes 
		              WHERE sender_id = $1 AND receiver_id = $2 AND period_id = $3)
	`, senderID, receiverID, periodID).Scan(&exists)
	return exists, err
}

// GetUserStats retrieves voting statistics for a user in a period
func (r *VoteRepository) GetUserStats(ctx context.Context, userID, periodID uuid.UUID) (*domain.UserStats, error) {
	var stats domain.UserStats
	var totalWeight int64

	err := r.pool.QueryRow(ctx, `
		SELECT u.id, CONCAT(u.first_name, ' ', COALESCE(u.last_name, '')),
		       COUNT(v.id), COALESCE(SUM(v.weight), 0)
		FROM users u
		LEFT JOIN votes v ON u.id = v.receiver_id AND v.period_id = $2
		WHERE u.id = $1
		GROUP BY u.id, u.first_name, u.last_name
	`, userID, periodID).Scan(&stats.UserID, &stats.UserName, &stats.TotalVotes, &totalWeight)
	if err != nil {
		return nil, err
	}
	stats.TotalWeight = int(totalWeight)

	return &stats, nil
}

// GetPeriodLeaderboard retrieves the leaderboard for a period
func (r *VoteRepository) GetPeriodLeaderboard(ctx context.Context, periodID uuid.UUID) ([]*domain.UserStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, CONCAT(u.first_name, ' ', COALESCE(u.last_name, '')),
		       COUNT(v.id), COALESCE(SUM(v.weight), 0)
		FROM users u
		LEFT JOIN votes v ON u.id = v.receiver_id AND v.period_id = $1
		WHERE u.is_active = TRUE
		GROUP BY u.id, u.first_name, u.last_name
		ORDER BY SUM(v.weight) DESC NULLS LAST, COUNT(v.id) DESC
	`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*domain.UserStats
	for rows.Next() {
		var s domain.UserStats
		var totalWeight int64
		err := rows.Scan(&s.UserID, &s.UserName, &s.TotalVotes, &totalWeight)
		if err != nil {
			return nil, err
		}
		s.TotalWeight = int(totalWeight)
		stats = append(stats, &s)
	}
	return stats, rows.Err()
}
