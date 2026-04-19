package usecase

import (
	"context"
	"errors"
	"time"

	"tg-stars-bot/internal/domain"

	"github.com/google/uuid"
)

// ErrForbidden is returned when action is not allowed
var ErrForbidden = errors.New("forbidden: insufficient role permissions")

// AdminUseCase handles admin operations (HR/Manager only)
type AdminUseCase struct {
	userRepo     domain.UserRepository
	periodRepo   domain.PeriodRepository
	voteRepo     domain.VoteRepository
	bitrixClient domain.BitrixClient
}

// NewAdminUseCase creates a new AdminUseCase
func NewAdminUseCase(
	userRepo domain.UserRepository,
	periodRepo domain.PeriodRepository,
	voteRepo domain.VoteRepository,
	bitrixClient domain.BitrixClient,
) *AdminUseCase {
	return &AdminUseCase{
		userRepo:     userRepo,
		periodRepo:   periodRepo,
		voteRepo:     voteRepo,
		bitrixClient: bitrixClient,
	}
}

// CheckHRAdmin checks if a user has HR or Manager role
func (uc *AdminUseCase) CheckHRAdmin(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if !user.IsHR() && !user.IsManager() {
		return ErrForbidden
	}
	return nil
}

// PeriodInput represents input for creating/updating a period
type PeriodInput struct {
	Name             string
	StartDate        time.Time
	EndDate          time.Time
	VotesPerEmployee int
	VoteWeight       int
}

// CreatePeriod creates a new voting period (HR only)
func (uc *AdminUseCase) CreatePeriod(ctx context.Context, adminID uuid.UUID, input PeriodInput) (*domain.Period, error) {
	if err := uc.CheckHRAdmin(ctx, adminID); err != nil {
		return nil, err
	}

	period := &domain.Period{
		Name:             input.Name,
		StartDate:        input.StartDate,
		EndDate:          input.EndDate,
		IsActive:         false,
		VotesPerEmployee: input.VotesPerEmployee,
		VoteWeight:       input.VoteWeight,
	}

	if period.VotesPerEmployee == 0 {
		period.VotesPerEmployee = 3
	}
	if period.VoteWeight == 0 {
		period.VoteWeight = 5
	}

	if err := uc.periodRepo.Create(ctx, period); err != nil {
		return nil, err
	}

	return period, nil
}

// OpenPeriod opens a period for voting (activates it)
func (uc *AdminUseCase) OpenPeriod(ctx context.Context, adminID, periodID uuid.UUID) (*domain.Period, error) {
	if err := uc.CheckHRAdmin(ctx, adminID); err != nil {
		return nil, err
	}

	if err := uc.periodRepo.SetActive(ctx, periodID); err != nil {
		return nil, err
	}

	return uc.periodRepo.GetByID(ctx, periodID)
}

// ClosePeriod closes a period (stops voting)
func (uc *AdminUseCase) ClosePeriod(ctx context.Context, adminID, periodID uuid.UUID) (*domain.Period, error) {
	if err := uc.CheckHRAdmin(ctx, adminID); err != nil {
		return nil, err
	}

	if err := uc.periodRepo.Close(ctx, periodID); err != nil {
		return nil, err
	}

	return uc.periodRepo.GetByID(ctx, periodID)
}

// SetUserVotingActive enables or disables voting for a user
func (uc *AdminUseCase) SetUserVotingActive(ctx context.Context, adminID, userID uuid.UUID, active bool) error {
	if err := uc.CheckHRAdmin(ctx, adminID); err != nil {
		return err
	}

	return uc.userRepo.SetVotingActive(ctx, userID, active)
}

// SetUserRole changes a user's role (Manager only)
func (uc *AdminUseCase) SetUserRole(ctx context.Context, adminID, userID uuid.UUID, role domain.UserRole) error {
	if err := uc.CheckHRAdmin(ctx, adminID); err != nil {
		return err
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	user.Role = role
	return uc.userRepo.Update(ctx, user)
}

// DeactivateUser deactivates a user
func (uc *AdminUseCase) DeactivateUser(ctx context.Context, adminID, userID uuid.UUID) error {
	if err := uc.CheckHRAdmin(ctx, adminID); err != nil {
		return err
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	user.IsActive = false
	return uc.userRepo.Update(ctx, user)
}

// SyncUsersFromBitrix synchronizes users from Bitrix24
func (uc *AdminUseCase) SyncUsersFromBitrix(ctx context.Context, adminID uuid.UUID) error {
	if err := uc.CheckHRAdmin(ctx, adminID); err != nil {
		return err
	}

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

// GetActivePeriod returns the currently active period
func (uc *AdminUseCase) GetActivePeriod(ctx context.Context) (*domain.Period, error) {
	return uc.periodRepo.GetActive(ctx)
}

// ListPeriods returns all periods
func (uc *AdminUseCase) ListPeriods(ctx context.Context) ([]*domain.Period, error) {
	return uc.periodRepo.List(ctx)
}

// ListUsers returns all active users
func (uc *AdminUseCase) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return uc.userRepo.List(ctx)
}

// ListHRAdmins returns all HR and Manager users
func (uc *AdminUseCase) ListHRAdmins(ctx context.Context) ([]*domain.User, error) {
	hrUsers, err := uc.userRepo.ListByRole(ctx, domain.RoleHR)
	if err != nil {
		return nil, err
	}
	managerUsers, err := uc.userRepo.ListByRole(ctx, domain.RoleManager)
	if err != nil {
		return nil, err
	}
	return append(hrUsers, managerUsers...), nil
}
