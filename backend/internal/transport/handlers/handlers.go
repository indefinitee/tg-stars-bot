package handlers

import (
	"net/http"

	"tg-stars-bot/internal/domain"
	"tg-stars-bot/internal/transport/middleware"
	"tg-stars-bot/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler holds all HTTP handlers
type Handler struct {
	voteUC   *usecase.VoteUseCase
	reportUC *usecase.ReportUseCase
	adminUC  *usecase.AdminUseCase
}

// NewHandler creates a new Handler
func NewHandler(
	voteUC *usecase.VoteUseCase,
	reportUC *usecase.ReportUseCase,
	adminUC *usecase.AdminUseCase,
) *Handler {
	return &Handler{
		voteUC:   voteUC,
		reportUC: reportUC,
		adminUC:  adminUC,
	}
}

// RegisterRoutes registers all routes
func (h *Handler) RegisterRoutes(r *gin.Engine, botToken string) {
	// Auth middleware
	auth := middleware.AuthMiddleware(botToken)

	// Public routes
	r.GET("/health", h.HealthCheck)

	// Protected routes
	api := r.Group("/api/v1")
	api.Use(auth)
	{
		// Vote endpoints
		api.POST("/vote", h.CastVote)
		api.GET("/votes/remaining", h.GetRemainingVotes)
		api.GET("/votes/my", h.GetMyVotes)

		// Report endpoints
		api.GET("/leaderboard", h.GetLeaderboard)
		api.GET("/stats/me", h.GetMyStats)

		// User endpoints
		api.GET("/users", h.ListUsers)
		api.GET("/users/:id", h.GetUser)
		api.GET("/periods/active", h.GetActivePeriod)
	}

	// Admin routes (HR/Manager only)
	admin := r.Group("/api/v1/admin")
	admin.Use(auth)
	admin.Use(middleware.HRAdminMiddleware(h.getUserRole))
	{
		// Period management
		admin.GET("/periods", h.ListPeriods)
		admin.POST("/periods", h.CreatePeriod)
		admin.POST("/periods/:id/open", h.OpenPeriod)
		admin.POST("/periods/:id/close", h.ClosePeriod)

		// User management
		admin.POST("/users/:id/voting", h.SetUserVoting)
		admin.POST("/users/:id/role", h.SetUserRole)
		admin.POST("/users/sync", h.SyncUsers)

		// Reports
		admin.GET("/periods/:id/report", h.GetPeriodReport)
		admin.GET("/periods/:id/leaderboard", h.GetPeriodLeaderboard)
		admin.GET("/periods/:id/votes", h.GetPeriodVotes)
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// VoteRequest represents a vote request body
type VoteRequest struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
	Weight     int    `json:"weight"`
	Message    string `json:"message"`
}

// CastVote handles vote creation
func (h *Handler) CastVote(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receiverID, err := uuid.Parse(req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid receiver_id"})
		return
	}

	vote, err := h.voteUC.CastVote(c.Request.Context(), usecase.VoteInput{
		SenderID:   userID,
		ReceiverID: receiverID,
		Weight:     req.Weight,
		Message:    req.Message,
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case usecase.ErrNoActivePeriod, usecase.ErrVotingNotAllowed,
			usecase.ErrNoVotesLeft, usecase.ErrAlreadyVoted,
			usecase.ErrCannotVoteSelf, usecase.ErrInvalidReceiver:
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, vote)
}

// GetRemainingVotes returns remaining votes for the current user
func (h *Handler) GetRemainingVotes(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	remaining, err := h.voteUC.GetRemainingVotes(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"remaining_votes": remaining})
}

// GetMyVotes returns all votes sent by the current user
func (h *Handler) GetMyVotes(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	votes, err := h.voteUC.GetMyVotes(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, votes)
}

// GetLeaderboard returns the leaderboard for the active period
func (h *Handler) GetLeaderboard(c *gin.Context) {
	leaderboard, err := h.reportUC.GetActivePeriodLeaderboard(c.Request.Context())
	if err != nil {
		if err == usecase.ErrNoActivePeriod {
			c.JSON(http.StatusOK, gin.H{"leaderboard": []interface{}{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
}

// GetMyStats returns statistics for the current user
func (h *Handler) GetMyStats(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.reportUC.GetActivePeriodStats(c.Request.Context(), userID)
	if err != nil {
		if err == usecase.ErrNoActivePeriod {
			c.JSON(http.StatusOK, gin.H{"stats": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListUsers returns all active users
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.adminUC.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser returns a user by ID
func (h *Handler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	users, err := h.adminUC.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, u := range users {
		if u.ID == id {
			c.JSON(http.StatusOK, u)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
}

// GetActivePeriod returns the currently active period
func (h *Handler) GetActivePeriod(c *gin.Context) {
	period, err := h.adminUC.GetActivePeriod(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if period == nil {
		c.JSON(http.StatusOK, gin.H{"period": nil})
		return
	}

	c.JSON(http.StatusOK, period)
}

// ListPeriods returns all periods
func (h *Handler) ListPeriods(c *gin.Context) {
	periods, err := h.adminUC.ListPeriods(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, periods)
}

// CreatePeriodRequest represents period creation request
type CreatePeriodRequest struct {
	Name             string `json:"name" binding:"required"`
	StartDate        string `json:"start_date" binding:"required"`
	EndDate          string `json:"end_date" binding:"required"`
	VotesPerEmployee int    `json:"votes_per_employee"`
	VoteWeight       int    `json:"vote_weight"`
}

// CreatePeriod creates a new period
func (h *Handler) CreatePeriod(c *gin.Context) {
	adminID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req CreatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := usecase.PeriodInput{
		Name:             req.Name,
		VotesPerEmployee: req.VotesPerEmployee,
		VoteWeight:       req.VoteWeight,
	}

	period, err := h.adminUC.CreatePeriod(c.Request.Context(), adminID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, period)
}

// OpenPeriod opens a period for voting
func (h *Handler) OpenPeriod(c *gin.Context) {
	adminID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	period, err := h.adminUC.OpenPeriod(c.Request.Context(), adminID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, period)
}

// ClosePeriod closes a period
func (h *Handler) ClosePeriod(c *gin.Context) {
	adminID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	period, err := h.adminUC.ClosePeriod(c.Request.Context(), adminID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, period)
}

// SetUserVotingRequest represents request to set user voting status
type SetUserVotingRequest struct {
	Active bool `json:"active"`
}

// SetUserVoting enables or disables voting for a user
func (h *Handler) SetUserVoting(c *gin.Context) {
	adminID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req SetUserVotingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.adminUC.SetUserVotingActive(c.Request.Context(), adminID, userID, req.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SetUserRoleRequest represents request to change user role
type SetUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// SetUserRole changes a user's role
func (h *Handler) SetUserRole(c *gin.Context) {
	adminID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req SetUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.adminUC.SetUserRole(c.Request.Context(), adminID, userID, domain.UserRole(req.Role)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncUsers synchronizes users from Bitrix24
func (h *Handler) SyncUsers(c *gin.Context) {
	adminID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.adminUC.SyncUsersFromBitrix(c.Request.Context(), adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetPeriodReport returns comprehensive report for a period
func (h *Handler) GetPeriodReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	report, err := h.reportUC.GetPeriodReport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetPeriodLeaderboard returns leaderboard for a period
func (h *Handler) GetPeriodLeaderboard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	leaderboard, err := h.reportUC.GetLeaderboard(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
}

// GetPeriodVotes returns all votes for a period
func (h *Handler) GetPeriodVotes(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	votes, err := h.reportUC.GetAllVotes(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, votes)
}

// getUserID retrieves user UUID from context
func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, error) {
	return middleware.GetUserIDFromContext(c, func(telegramID int64) (uuid.UUID, error) {
		// This would be implemented with a callback passed to the handler
		return uuid.Nil, nil
	})
}

// getUserRole retrieves user role from Telegram ID
func (h *Handler) getUserRole(telegramID int64) (string, error) {
	return "", nil
}
