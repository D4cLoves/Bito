package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"bito/internal/model"

	"github.com/google/uuid"
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	user.PasswordHash = ""

	return user, nil
}

func (s *UserService) UpdateUsername(ctx context.Context, userID uuid.UUID, newUsername string) (*model.User, error) {
	newUsername = strings.TrimSpace(newUsername)
	if n := utf8.RuneCountInString(newUsername); n < minUsernameLength || n > maxUsernameLength {
		return nil, ErrInvalidUsername
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("update username: get user: %w", err)
	}

	user.Name = newUsername

	if err := s.repo.Update(ctx, user); err != nil {
		if errors.Is(err, model.ErrUserAlreadyExists) {
			return nil, model.ErrUserAlreadyExists
		}
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("update username: %w", err)
	}

	user.PasswordHash = ""

	return user, nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if len(newPassword) > maxPasswordLength {
		return ErrPasswordTooLong
	}
	if oldPassword == newPassword {
		return ErrSamePassword
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return model.ErrUserNotFound
		}
		return fmt.Errorf("change password: get user: %w", err)
	}

	if !CheckPassword(oldPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("change password: hash: %w", err)
	}

	user.PasswordHash = newHash

	if err := s.repo.Update(ctx, user); err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return model.ErrUserNotFound
		}
		return fmt.Errorf("change password: %w", err)
	}

	return nil
}

func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return model.ErrUserNotFound
		}
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}
