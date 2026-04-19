package usecase

import (
	"context"
	"errors"

	"tg-stars-bot/internal/domain"

	"github.com/google/uuid"
)

// ErrUnauthorized is returned when a user doesn't have permission
var ErrUnauthorized = errors.New("unauthorized: insufficient permissions")

// ReportUseCase handles reporting business logic
type ReportUseCase struct {
	userRepo   domain.UserRepository
	periodRepo domain.PeriodRepository
	voteRepo   domain.VoteRepository
}

// NewReportUseCase creates a new ReportUseCase
func NewReportUseCase(
	userRepo domain.UserRepository,
	periodRepo domain.PeriodRepository,
	voteRepo domain.VoteRepository,
) *ReportUseCase {
	return &ReportUseCase{
		userRepo:   userRepo,
		periodRepo: periodRepo,
		voteRepo:   voteRepo,
	}
}

// GetLeaderboard returns the leaderboard for a period
func (uc *ReportUseCase) GetLeaderboard(ctx context.Context, periodID uuid.UUID) ([]*domain.UserStats, error) {
	return uc.voteRepo.GetPeriodLeaderboard(ctx, periodID)
}

// GetActivePeriodLeaderboard returns the leaderboard for the active period
func (uc *ReportUseCase) GetActivePeriodLeaderboard(ctx context.Context) ([]*domain.UserStats, error) {
	period, err := uc.periodRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if period == nil {
		return nil, ErrNoActivePeriod
	}

	return uc.voteRepo.GetPeriodLeaderboard(ctx, period.ID)
}

// GetUserStats returns statistics for a specific user in a period
func (uc *ReportUseCase) GetUserStats(ctx context.Context, userID, periodID uuid.UUID) (*domain.UserStats, error) {
	return uc.voteRepo.GetUserStats(ctx, userID, periodID)
}

// GetActivePeriodStats returns statistics for a user in the active period
func (uc *ReportUseCase) GetActivePeriodStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	period, err := uc.periodRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if period == nil {
		return nil, ErrNoActivePeriod
	}

	return uc.voteRepo.GetUserStats(ctx, userID, period.ID)
}

// GetUserVotesReceived returns all votes received by a user in a period
func (uc *ReportUseCase) GetUserVotesReceived(ctx context.Context, userID, periodID uuid.UUID) ([]*domain.VoteWithUsers, error) {
	return uc.voteRepo.ListByReceiver(ctx, userID, periodID)
}

// GetAllVotes returns all votes for a period
func (uc *ReportUseCase) GetAllVotes(ctx context.Context, periodID uuid.UUID) ([]*domain.Vote, error) {
	return uc.voteRepo.ListByPeriod(ctx, periodID)
}

// PeriodStats represents aggregated statistics for a period
type PeriodStats struct {
	PeriodID    uuid.UUID
	PeriodName  string
	TotalVotes  int
	TotalWeight int
	ActiveUsers int
	VotedUsers  int
	Leaderboard []*domain.UserStats
}

// GetPeriodReport returns comprehensive statistics for a period
func (uc *ReportUseCase) GetPeriodReport(ctx context.Context, periodID uuid.UUID) (*PeriodStats, error) {
	period, err := uc.periodRepo.GetByID(ctx, periodID)
	if err != nil {
		return nil, err
	}
	if period == nil {
		return nil, errors.New("period not found")
	}

	leaderboard, err := uc.voteRepo.GetPeriodLeaderboard(ctx, periodID)
	if err != nil {
		return nil, err
	}

	votes, err := uc.voteRepo.ListByPeriod(ctx, periodID)
	if err != nil {
		return nil, err
	}

	users, err := uc.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	var totalWeight int
	votedUsers := make(map[uuid.UUID]bool)
	for _, v := range votes {
		totalWeight += v.Weight
		votedUsers[v.SenderID] = true
	}

	stats := &PeriodStats{
		PeriodID:    period.ID,
		PeriodName:  period.Name,
		TotalVotes:  len(votes),
		TotalWeight: totalWeight,
		ActiveUsers: len(users),
		VotedUsers:  len(votedUsers),
		Leaderboard: leaderboard,
	}

	return stats, nil
}
