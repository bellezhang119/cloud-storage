package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bellezhang119/cloud-storage/internal/database"
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
	queries     Queries
	userService UserGetter
}

func NewService(q Queries, us UserGetter) *Service {
	return &Service{
		queries:     q,
		userService: us,
	}
}

func (s *Service) CreateUser(ctx context.Context, email, password string) (database.User, error) {
	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return database.User{}, err
	}

	verificationToken, err := util.GenerateVerificationToken()
	if err != nil {
		return database.User{}, err
	}

	expiry := time.Now().Add(24 * time.Hour)

	return s.queries.CreateUser(ctx, database.CreateUserParams{
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
}

func (s *Service) VerifyUserByToken(ctx context.Context, token string) error {
	user, err := s.queries.GetUserByVerificationToken(ctx, sql.NullString{
		String: token,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("invalid or expired token")
		}
		return err
	}

	if !user.VerificationTokenExpiry.Valid || user.VerificationTokenExpiry.Time.Before(time.Now()) {
		return errors.New("token has expired")
	}

	rowsAffected, err := s.queries.MarkUserAsVerified(ctx, user.ID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("failed to verify user: no rows updated")
	}

	return nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	return s.userService.GetUserByEmail(ctx, email)
}

func (s *Service) UpdateVerificationToken(ctx context.Context, user database.User) (string, error) {
	verificationToken, err := util.GenerateVerificationToken()
	if err != nil {
		return "", err
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
		return "", err
	}
	if rowsAffected == 0 {
		return "", errors.New("failed to update verification token: no rows updated")
	}

	return verificationToken, nil
}

func (s *Service) AuthenticateUser(ctx context.Context, email, password string) (database.User, error) {
	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		return database.User{}, err
	}

	if err := util.CheckPassword(user.PasswordHash, password); err != nil {
		return database.User{}, err
	}

	return user, nil
}

func (s *Service) GenerateJWTTokens(
	ctx context.Context,
	user database.User,
) (accessToken string, refreshToken string, err error) {
	expiry := time.Now().Add(expireTime * 24 * time.Hour)

	accessToken, refreshToken, err = util.GenerateJWTTokens(user.ID, user.Email, expiry)
	if err != nil {
		return "", "", err
	}

	hashedRefreshToken := util.HashToken(refreshToken)

	tokenRow, err := s.queries.InsertRefreshToken(ctx, database.InsertRefreshTokenParams{
		TokenHash: hashedRefreshToken,
		UserID:    user.ID,
		ExpiresAt: expiry,
	})
	if err != nil {
		return "", "", err
	}

	_ = tokenRow

	return accessToken, refreshToken, nil
}

func (s *Service) RefreshJWTTokens(ctx context.Context, oldRefreshToken string) (accessToken string, refreshToken string, err error) {
	claims, err := util.VerifyRefreshToken(oldRefreshToken)
	if err != nil {
		return "", "", err
	}

	userID := int32(claims["user_id"].(float64))
	hashedOld := util.HashToken(oldRefreshToken)

	rt, err := s.queries.GetRefreshToken(ctx, hashedOld)
	if err != nil || rt.Revoked {
		return "", "", errors.New("refresh token revoked or not found")
	}

	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", err
	}

	expiry := time.Unix(int64(claims["exp"].(float64)), 0)
	accessToken, refreshToken, err = util.GenerateJWTTokens(user.ID, user.Email, expiry)
	if err != nil {
		return "", "", err
	}

	newHashed := util.HashToken(refreshToken)

	_, err = s.queries.InsertRefreshToken(ctx, database.InsertRefreshTokenParams{
		TokenHash: newHashed,
		UserID:    user.ID,
		ExpiresAt: expiry,
	})
	if err != nil {
		return "", "", err
	}

	rowsAffected, err := s.queries.RevokeRefreshToken(ctx, hashedOld)
	if err != nil {
		return "", "", err
	}
	if rowsAffected == 0 {
		return "", "", errors.New("failed to revoke old refresh token: no rows updated")
	}

	return accessToken, refreshToken, nil
}
