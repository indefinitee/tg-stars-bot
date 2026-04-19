package usecase

import (
	"context"
	"errors"
	"fmt"

	"tg-stars-bot/internal/domain"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrNoActivePeriod   = errors.New("no active voting period")
	ErrVotingNotAllowed = errors.New("voting is not allowed for this user")
	ErrNoVotesLeft      = errors.New("no votes left for this period")
	ErrAlreadyVoted     = errors.New("already voted for this user in this period")
	ErrCannotVoteSelf   = errors.New("cannot vote for yourself")
	ErrInvalidReceiver  = errors.New("invalid receiver")
	ErrPeriodNotActive  = errors.New("period is not active")
)

// VoteUseCase handles voting business logic
type VoteUseCase struct {
	userRepo     domain.UserRepository
	periodRepo   domain.PeriodRepository
	voteRepo     domain.VoteRepository
	bitrixClient domain.BitrixClient
}

// NewVoteUseCase creates a new VoteUseCase
func NewVoteUseCase(
	userRepo domain.UserRepository,
	periodRepo domain.PeriodRepository,
	voteRepo domain.VoteRepository,
	bitrixClient domain.BitrixClient,
) *VoteUseCase {
	return &VoteUseCase{
		userRepo:     userRepo,
		periodRepo:   periodRepo,
		voteRepo:     voteRepo,
		bitrixClient: bitrixClient,
	}
}

// VoteInput represents input for casting a vote
type VoteInput struct {
	SenderID   uuid.UUID `json:"sender_id"`
	ReceiverID uuid.UUID `json:"receiver_id"`
	Weight     int       `json:"weight"`
	Message    string    `json:"message"`
}

// CastVote creates a new vote with business rule validation
func (uc *VoteUseCase) CastVote(ctx context.Context, input VoteInput) (*domain.Vote, error) {
	// Get active period
	period, err := uc.periodRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if period == nil {
		return nil, ErrNoActivePeriod
	}

	// Get sender
	sender, err := uc.userRepo.GetByID(ctx, input.SenderID)
	if err != nil {
		return nil, err
	}
	if sender == nil {
		return nil, fmt.Errorf("sender not found")
	}

	// Check if sender can vote
	if !sender.CanVote() {
		return nil, ErrVotingNotAllowed
	}

	// Get receiver
	receiver, err := uc.userRepo.GetByID(ctx, input.ReceiverID)
	if err != nil {
		return nil, err
	}
	if receiver == nil {
		return nil, ErrInvalidReceiver
	}

	// Check if receiver can receive votes
	if !receiver.IsActive {
		return nil, ErrInvalidReceiver
	}

	// Cannot vote for self
	if input.SenderID == input.ReceiverID {
		return nil, ErrCannotVoteSelf
	}

	// Check if already voted for this user
	hasVoted, err := uc.voteRepo.HasVotedFor(ctx, input.SenderID, input.ReceiverID, period.ID)
	if err != nil {
		return nil, err
	}
	if hasVoted {
		return nil, ErrAlreadyVoted
	}

	// Count remaining votes
	voteCount, err := uc.voteRepo.CountBySender(ctx, input.SenderID, period.ID)
	if err != nil {
		return nil, err
	}
	if voteCount >= period.VotesPerEmployee {
		return nil, ErrNoVotesLeft
	}

	// Set default weight
	weight := input.Weight
	if weight <= 0 {
		weight = period.VoteWeight
	}

	// Create vote
	vote := &domain.Vote{
		SenderID:   input.SenderID,
		ReceiverID: input.ReceiverID,
		PeriodID:   period.ID,
		Weight:     weight,
		Message:    input.Message,
	}

	if err := uc.voteRepo.Create(ctx, vote); err != nil {
		return nil, err
	}

	return vote, nil
}

// GetRemainingVotes returns the number of remaining votes for a user in the active period
func (uc *VoteUseCase) GetRemainingVotes(ctx context.Context, userID uuid.UUID) (int, error) {
	period, err := uc.periodRepo.GetActive(ctx)
	if err != nil {
		return 0, err
	}
	if period == nil {
		return 0, ErrNoActivePeriod
	}

	voteCount, err := uc.voteRepo.CountBySender(ctx, userID, period.ID)
	if err != nil {
		return 0, err
	}

	return period.VotesPerEmployee - voteCount, nil
}

// GetMyVotes returns all votes sent by a user in the active period
func (uc *VoteUseCase) GetMyVotes(ctx context.Context, userID uuid.UUID) ([]*domain.Vote, error) {
	period, err := uc.periodRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if period == nil {
		return nil, ErrNoActivePeriod
	}

	return uc.voteRepo.ListBySender(ctx, userID, period.ID)
}

// SyncFromBitrix synchronizes users from Bitrix24
func (uc *VoteUseCase) SyncFromBitrix(ctx context.Context) error {
	users, err := uc.bitrixClient.GetEmployees(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		if err := uc.userRepo.UpsertFromBitrix(ctx, user); err != nil {
			return err
		}
	}

	return nil
}
