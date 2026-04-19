package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// GetByID retrieves a user by their UUID
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByTelegramID retrieves a user by their Telegram ID
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)

	// GetByBitrixID retrieves a user by their Bitrix24 ID
	GetByBitrixID(ctx context.Context, bitrixID int) (*User, error)

	// List retrieves all active users
	List(ctx context.Context) ([]*User, error)

	// ListByRole retrieves users by role
	ListByRole(ctx context.Context, role UserRole) ([]*User, error)

	// Create creates a new user
	Create(ctx context.Context, user *User) error

	// Update updates an existing user
	Update(ctx context.Context, user *User) error

	// UpsertFromBitrix creates or updates a user from Bitrix24 data
	UpsertFromBitrix(ctx context.Context, user *User) error

	// SetVotingActive sets whether a user can vote
	SetVotingActive(ctx context.Context, id uuid.UUID, active bool) error
}

// PeriodRepository defines the interface for period data access
type PeriodRepository interface {
	// GetByID retrieves a period by its UUID
	GetByID(ctx context.Context, id uuid.UUID) (*Period, error)

	// GetActive retrieves the currently active period
	GetActive(ctx context.Context) (*Period, error)

	// List retrieves all periods
	List(ctx context.Context) ([]*Period, error)

	// Create creates a new period
	Create(ctx context.Context, period *Period) error

	// Update updates an existing period
	Update(ctx context.Context, period *Period) error

	// SetActive sets the period as active (deactivates all others)
	SetActive(ctx context.Context, id uuid.UUID) error

	// Close closes a period (makes it inactive)
	Close(ctx context.Context, id uuid.UUID) error
}

// VoteRepository defines the interface for vote data access
type VoteRepository interface {
	// Create creates a new vote
	Create(ctx context.Context, vote *Vote) error

	// GetByID retrieves a vote by its UUID
	GetByID(ctx context.Context, id uuid.UUID) (*Vote, error)

	// ListByPeriod retrieves all votes for a period
	ListByPeriod(ctx context.Context, periodID uuid.UUID) ([]*Vote, error)

	// ListBySender retrieves all votes sent by a user in a period
	ListBySender(ctx context.Context, senderID, periodID uuid.UUID) ([]*Vote, error)

	// ListByReceiver retrieves all votes received by a user in a period
	ListByReceiver(ctx context.Context, receiverID, periodID uuid.UUID) ([]*VoteWithUsers, error)

	// CountBySender counts votes sent by a user in a period
	CountBySender(ctx context.Context, senderID, periodID uuid.UUID) (int, error)

	// HasVotedFor checks if a user has already voted for another user in a period
	HasVotedFor(ctx context.Context, senderID, receiverID, periodID uuid.UUID) (bool, error)

	// GetUserStats retrieves voting statistics for a user in a period
	GetUserStats(ctx context.Context, userID, periodID uuid.UUID) (*UserStats, error)

	// GetPeriodLeaderboard retrieves the leaderboard for a period
	GetPeriodLeaderboard(ctx context.Context, periodID uuid.UUID) ([]*UserStats, error)
}

// BitrixClient defines the interface for Bitrix24 API client
type BitrixClient interface {
	// GetEmployees retrieves all employees from Bitrix24
	GetEmployees(ctx context.Context) ([]*User, error)

	// SyncUsers synchronizes users from Bitrix24 to the local database
	SyncUsers(ctx context.Context) error
}
