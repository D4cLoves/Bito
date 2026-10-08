package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
	minUsernameLength = 3
	maxUsernameLength = 32
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password cannot exceed 72 bytes")
	ErrInvalidUsername    = errors.New("username must be between 3 and 32 characters")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrSamePassword       = errors.New("new password cannot be the same as old password")
	ErrGeneratePassword   = errors.New("invalid generate code error")
	ErrInvalidCode        = errors.New("invalid code")
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User, stats *model.UserStats, wallet *model.UserWallet) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type CodeProvider interface {
	GenerateNumericOTP() (string, error)
}

type MailerProvider interface {
	SendVerificationCode(ctx context.Context, toEmail, code string) error
}

type RedisProvider interface {
	SavePending(ctx context.Context, data *model.PendingRegistration, ttl time.Duration) error
	VerifyAndGetPending(ctx context.Context, email, code string) (*model.PendingRegistration, error)
	SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string, ttl time.Duration) error
	ValidateAndRevokeRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error)
	RevokeRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string) error
}


type AuthService struct {
	repo  UserRepository
	otp   CodeProvider
	mail  MailerProvider
	redis RedisProvider
}

func NewAuthService(repo UserRepository, otp CodeProvider, mail MailerProvider, redis RedisProvider) *AuthService {
	return &AuthService{
		repo:  repo,
		otp:   otp,
		mail:  mail,
		redis: redis,
	}
}

func (s *AuthService) SendCode(ctx context.Context, email, username, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	if email == "" || !strings.Contains(email, "@") {
		return ErrInvalidEmail
	}

	if len(username) < minUsernameLength || len(username) > maxUsernameLength {
		return ErrInvalidUsername
	}

	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLength {
		return ErrPasswordTooLong
	}

	// 1. Проверяем, существует ли пользователь в базе
	existingUser, err := s.repo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, model.ErrUserNotFound) {
		return fmt.Errorf("check existing email: %w", err)
	}
	if existingUser != nil {
		return model.ErrUserAlreadyExists
	}

	// 2. Хешируем пароль
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// 3. Генерируем 6-значный OTP код
	code, err := s.otp.GenerateNumericOTP()
	if err != nil {
		return ErrGeneratePassword
	}

	// 4. Сохраняем в Redis с правильным TTL (15 минут)1111
	pending := &model.PendingRegistration{
		Code:         code,
		Email:        email,
		Name:         username,
		PasswordHash: hashedPassword,
	}

	if err := s.redis.SavePending(ctx, pending, 15*time.Minute); err != nil {
		return fmt.Errorf("save pending registration in redis: %w", err)
	}

	// 5. Отправляем письмо с кодом
	if err := s.mail.SendVerificationCode(ctx, email, code); err != nil {
		return fmt.Errorf("send verification code: %w", err)
	}

	return nil
}

func (s *AuthService) VerifyAndRegister(ctx context.Context, email, code string) (*model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)

	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrInvalidEmail
	}
	if len(code) != 6 {
		return nil, ErrInvalidCode
	}

	pending, err := s.redis.VerifyAndGetPending(ctx, email, code)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        pending.Email,
		Name:         pending.Name,
		PasswordHash: pending.PasswordHash,
	}

	stats := &model.UserStats{}
	wallet := &model.UserWallet{}

	if err := s.repo.CreateUser(ctx, user, stats, wallet); err != nil {
		if errors.Is(err, model.ErrUserAlreadyExists) {
			return nil, model.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}





func (s *AuthService) LoginUser(ctx context.Context, email, password string) (*model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("login user: %w", err)
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	user.PasswordHash = ""

	return user, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
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

func (s *AuthService) UpdateUsername(ctx context.Context, userID uuid.UUID, newUsername string) (*model.User, error) {
	newUsername = strings.TrimSpace(newUsername)
	if len(newUsername) < minUsernameLength || len(newUsername) > maxUsernameLength {
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

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
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

func (s *AuthService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return model.ErrUserNotFound
		}
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

func (s *AuthService) SaveSession(ctx context.Context, userID uuid.UUID, tokenID string, ttl time.Duration) error {
	return s.redis.SaveRefreshToken(ctx, userID, tokenID, ttl)
}

func (s *AuthService) ValidateAndRevokeSession(ctx context.Context, userID uuid.UUID, tokenID string) error {
	valid, err := s.redis.ValidateAndRevokeRefreshToken(ctx, userID, tokenID)
	if err != nil {
		return fmt.Errorf("validate session: %w", err)
	}
	if !valid {
		return model.ErrSessionExpired
	}
	return nil
}

func (s *AuthService) RevokeSession(ctx context.Context, userID uuid.UUID, tokenID string) error {
	return s.redis.RevokeRefreshToken(ctx, userID, tokenID)
}

func HashPassword(password string) (string, error) {
	passByte := []byte(password)

	hashedBytes, err := bcrypt.GenerateFromPassword(passByte, bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hashedBytes), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
