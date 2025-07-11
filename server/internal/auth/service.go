package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bellezhang119/cloud-storage/internal/database"
	"github.com/bellezhang119/cloud-storage/internal/util"
)

type Service struct {
	queries *database.Queries
}

func NewService(q *database.Queries) *Service {
	return &Service{queries: q}
}

func (s *Service) CreateUser(ctx context.Context, email, passwordHash string) (database.User, error) {
	verificationToken, err := util.GenerateVerificationToken()
	if err != nil {
		return database.User{}, err
	}

	expiry := time.Now().Add(24 * time.Hour)

	return s.queries.CreateUser(ctx, database.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
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
			return errors.New("Invalid or expired token")
		}
		return err
	}

	if !user.VerificationTokenExpiry.Valid || user.VerificationTokenExpiry.Time.Before(time.Now()) {
		return errors.New("Token has expired")
	}

	return s.queries.MarkUserAsVerified(ctx, user.ID)
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	return s.queries.GetUserByEmail(ctx, email)
}
