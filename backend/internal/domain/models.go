package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	RoleEmployee UserRole = "employee"
	RoleHR       UserRole = "hr"
	RoleManager  UserRole = "manager"
)

// User represents an employee in the system
type User struct {
	ID             uuid.UUID
	BitrixID       int
	TelegramID     int64
	Username       string
	FirstName      string
	LastName       string
	Email          string
	Role           UserRole
	IsActive       bool
	IsVotingActive bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CanVote returns true if the user can participate in voting
func (u *User) CanVote() bool {
	return u.IsActive && u.IsVotingActive
}

// IsHR returns true if the user has HR role
func (u *User) IsHR() bool {
	return u.Role == RoleHR
}

// IsManager returns true if the user has Manager role
func (u *User) IsManager() bool {
	return u.Role == RoleManager
}

// Period represents a voting period
type Period struct {
	ID               uuid.UUID
	Name             string
	StartDate        time.Time
	EndDate          time.Time
	IsActive         bool
	VotesPerEmployee int
	VoteWeight       int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsCurrent returns true if the current time is within the period
func (p *Period) IsCurrent() bool {
	now := time.Now()
	return now.After(p.StartDate) && now.Before(p.EndDate)
}

// Vote represents a star/recognition given from one user to another
type Vote struct {
	ID         uuid.UUID
	SenderID   uuid.UUID
	ReceiverID uuid.UUID
	PeriodID   uuid.UUID
	Weight     int
	Message    string
	CreatedAt  time.Time
}

// VoteWithUsers includes vote data along with sender and receiver user info
type VoteWithUsers struct {
	Vote
	SenderName   string
	ReceiverName string
}

// Stats represents voting statistics for a user within a period
type UserStats struct {
	UserID       uuid.UUID
	UserName     string
	TotalVotes   int
	TotalWeight  int
	ReceivedFrom []*VoteWithUsers
}
