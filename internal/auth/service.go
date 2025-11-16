package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/middleware"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

var expireTime time.Duration = 30

type Queries interface {
	CreateUser(ctx context.Context, params database.CreateUserParams) (database.User, error)
	GetUserByVerificationToken(ctx context.Context, token sql.NullString) (database.User, error)
	MarkUserAsVerified(ctx context.Context, id int32) (int64, error)
	UpdateVerificationToken(ctx context.Context, arg database.UpdateVerificationTokenParams) (int64, error)
	InsertRefreshToken(ctx context.Context, arg database.InsertRefreshTokenParams) (database.RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash string) (database.GetRefreshTokenRow, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) (int64, error)
}

type UserGetter interface {
	GetUserByID(ctx context.Context, id int32) (database.User, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
}

type Service struct {
	queries    Queries
	userGetter UserGetter
}

func NewService(q Queries, ug UserGetter) *Service {
	return &Service{
		queries:    q,
		userGetter: ug,
	}
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	return s.userGetter.GetUserByEmail(ctx, email)
}

func (s *Service) CreateUser(ctx context.Context, email, password string) (database.User, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("creating user")

	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		return database.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	verificationToken, err := util.GenerateVerificationToken()
	if err != nil {
		logger.Error("failed to generate verification token", "error", err)
		return database.User{}, fmt.Errorf("failed to generate verification token: %w", err)
	}

	expiry := time.Now().Add(24 * time.Hour)
	logger.Info("generated verification token", "expiry", expiry.Format(time.RFC3339))

	user, err := s.queries.CreateUser(ctx, database.CreateUserParams{
		Email:        email,
		PasswordHash: hashedPassword,
		IsVerified:   false,
		VerificationToken: sql.NullString{
			String: verificationToken,
			Valid:  true,
		},
		VerificationTokenExpiry: sql.NullTime{
			Time:  expiry,
			Valid: true,
		},
	})
	if err != nil {
		logger.Error("failed to create user in database", "error", err)
		return database.User{}, fmt.Errorf("failed to create user in database: %w", err)
	}

	logger.Info("user successfully created", "user_id", user.ID)
	return user, nil
}

func (s *Service) VerifyUserByToken(ctx context.Context, token string) error {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting user verification")

	user, err := s.queries.GetUserByVerificationToken(ctx, sql.NullString{
		String: token,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Warn("invalid or expired verification token")
			return fmt.Errorf("invalid or expired token: %w", err)
		}
		logger.Error("failed to retrieve user by verification token", "error", err)
		return fmt.Errorf("failed to get user by verification token: %w", err)
	}

	if !user.VerificationTokenExpiry.Valid || user.VerificationTokenExpiry.Time.Before(time.Now()) {
		logger.Warn("token has expired", "user_id", user.ID)
		return fmt.Errorf("token expired for user %d", user.ID)
	}

	rowsAffected, err := s.queries.MarkUserAsVerified(ctx, user.ID)
	if err != nil {
		logger.Error("failed to mark user as verified", "user_id", user.ID, "error", err)
		return fmt.Errorf("failed to mark user %d as verified: %w", user.ID, err)
	}
	if rowsAffected == 0 {
		logger.Error("no rows updated when verifying user", "user_id", user.ID)
		return fmt.Errorf("failed to verify user %d: no rows updated", user.ID)
	}

	logger.Info("user verified successfully", "user_id", user.ID)
	return nil
}

func (s *Service) UpdateVerificationToken(ctx context.Context, user database.User) (string, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting verification token update")

	verificationToken, err := util.GenerateVerificationToken()
	if err != nil {
		logger.Error("failed to generate verification token", "error", err)
		return "", fmt.Errorf("failed to generate verification token: %w", err)
	}

	expiry := time.Now().Add(24 * time.Hour)
	rowsAffected, err := s.queries.UpdateVerificationToken(ctx, database.UpdateVerificationTokenParams{
		VerificationToken: sql.NullString{
			String: verificationToken,
			Valid:  true,
		},
		VerificationTokenExpiry: sql.NullTime{
			Time:  expiry,
			Valid: true,
		},
		Email: user.Email,
	})
	if err != nil {
		logger.Error("failed to update verification token in database", "error", err)
		return "", fmt.Errorf("failed to update verification token for user %s: %w", user.Email, err)
	}

	if rowsAffected == 0 {
		logger.Warn("no rows updated when updating verification token")
		return "", fmt.Errorf("failed to update verification token for user %s: no rows updated", user.Email)
	}

	logger.Info("verification token updated successfully", "expires_at", expiry)
	return verificationToken, nil
}

func (s *Service) AuthenticateUser(ctx context.Context, email, password string) (database.User, error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting user authentication")

	user, err := s.userGetter.GetUserByEmail(ctx, email)
	if err != nil {
		logger.Error("failed to get user by email", "error", err)
		return database.User{}, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}

	if err := util.CheckPassword(user.PasswordHash, password); err != nil {
		logger.Warn("invalid password attempt")
		return database.User{}, fmt.Errorf("invalid password for user %s: %w", email, err)
	}

	logger.Info("user authenticated successfully")
	return user, nil
}

func (s *Service) GenerateJWTTokens(
	ctx context.Context,
	user database.User,
) (accessToken string, refreshToken string, err error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting JWT token generation")

	expiry := time.Now().Add(expireTime * 24 * time.Hour)

	accessToken, refreshToken, err = util.GenerateJWTTokens(user.ID, user.Email, expiry)
	if err != nil {
		logger.Error("failed to generate JWT tokens", "error", err)
		return "", "", fmt.Errorf("failed to generate JWT tokens: %w", err)
	}

	hashedRefreshToken := util.HashToken(refreshToken)
	_, err = s.queries.InsertRefreshToken(ctx, database.InsertRefreshTokenParams{
		TokenHash: hashedRefreshToken,
		UserID:    user.ID,
		ExpiresAt: expiry,
	})
	if err != nil {
		logger.Error("failed to insert refresh token into database", "error", err)
		return "", "", fmt.Errorf("failed to insert refresh token for user %d: %w", user.ID, err)
	}

	logger.Info("JWT tokens generated and refresh token stored", "expires_at", expiry)
	return accessToken, refreshToken, nil
}

func (s *Service) RefreshJWTTokens(ctx context.Context, oldRefreshToken string) (accessToken string, refreshToken string, err error) {
	logger := middleware.GetLogger(ctx)
	logger.Info("starting JWT token refresh")

	claims, err := util.VerifyRefreshToken(oldRefreshToken)
	if err != nil {
		logger.Warn("invalid or expired refresh token", "error", err)
		return "", "", fmt.Errorf("invalid or expired refresh token: %w", err)
	}

	userID := int32(claims["user_id"].(float64))
	logger = logger.With("user_id", userID)

	hashedOld := util.HashToken(oldRefreshToken)
	rt, err := s.queries.GetRefreshToken(ctx, hashedOld)
	if err != nil || rt.Revoked {
		logger.Warn("refresh token revoked or not found", "error", err)
		return "", "", fmt.Errorf("refresh token revoked or not found for user %d", userID)
	}

	user, err := s.userGetter.GetUserByID(ctx, userID)
	if err != nil {
		logger.Error("failed to fetch user by ID", "error", err)
		return "", "", fmt.Errorf("failed to fetch user %d: %w", userID, err)
	}

	expiry := time.Unix(int64(claims["exp"].(float64)), 0)
	accessToken, refreshToken, err = util.GenerateJWTTokens(user.ID, user.Email, expiry)
	if err != nil {
		logger.Error("failed to generate new JWT tokens", "error", err)
		return "", "", fmt.Errorf("failed to generate new JWT tokens: %w", err)
	}

	newHashed := util.HashToken(refreshToken)
	_, err = s.queries.InsertRefreshToken(ctx, database.InsertRefreshTokenParams{
		TokenHash: newHashed,
		UserID:    user.ID,
		ExpiresAt: expiry,
	})
	if err != nil {
		logger.Error("failed to insert new refresh token into database", "error", err)
		return "", "", fmt.Errorf("failed to insert new refresh token: %w", err)
	}

	rowsAffected, err := s.queries.RevokeRefreshToken(ctx, hashedOld)
	if err != nil {
		logger.Error("failed to revoke old refresh token", "error", err)
		return "", "", fmt.Errorf("failed to revoke old refresh token: %w", err)
	}
	if rowsAffected == 0 {
		logger.Warn("no rows updated when revoking old refresh token")
		return "", "", fmt.Errorf("failed to revoke old refresh token: no rows updated")
	}

	logger.Info("JWT tokens refreshed successfully", "expires_at", expiry)
	return accessToken, refreshToken, nil
}
