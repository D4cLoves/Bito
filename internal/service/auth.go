package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"bito/internal/model"
	"bito/internal/pkg/jwt"

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

type Session struct {
	User   *model.User
	Tokens *jwt.TokenPair
}

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

type TokenProvider interface {
	GenerateTokenPair(userID uuid.UUID, username string) (*jwt.TokenPair, error)
	ValidateRefreshToken(tokenString string) (*jwt.Claims, error)
}

type AuthService struct {
	repo       UserRepository
	otp        CodeProvider
	mail       MailerProvider
	redis      RedisProvider
	tokens     TokenProvider
	refreshTTL time.Duration
}

func NewAuthService(repo UserRepository, otp CodeProvider, mail MailerProvider, redis RedisProvider, tokens TokenProvider, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		repo:       repo,
		otp:        otp,
		mail:       mail,
		redis:      redis,
		tokens:     tokens,
		refreshTTL: refreshTTL,
	}
}

func (s *AuthService) SendCode(ctx context.Context, email, username, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	if email == "" || !strings.Contains(email, "@") {
		return ErrInvalidEmail
	}

	if n := utf8.RuneCountInString(username); n < minUsernameLength || n > maxUsernameLength {
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

func (s *AuthService) VerifyAndRegister(ctx context.Context, email, code string) (*Session, error) {
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

	user.PasswordHash = ""

	return s.issueSession(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*Session, error) {
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

	return s.issueSession(ctx, user)
}

func (s *AuthService) validateAndRevokeSession(ctx context.Context, userID uuid.UUID, tokenID string) error {
	valid, err := s.redis.ValidateAndRevokeRefreshToken(ctx, userID, tokenID)
	if err != nil {
		return fmt.Errorf("validate session: %w", err)
	}
	if !valid {
		return model.ErrSessionExpired
	}
	return nil
}

func (s *AuthService) issueSession(ctx context.Context, user *model.User) (*Session, error) {
	tokens, err := s.tokens.GenerateTokenPair(user.ID, user.Name)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}
	if err := s.redis.SaveRefreshToken(ctx, user.ID, tokens.TokenID, s.refreshTTL); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}
	return &Session{User: user, Tokens: tokens}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("validate refresh token: %w", err)
	}
	if err := s.validateAndRevokeSession(ctx, claims.UserID, claims.ID); err != nil {
		return nil, err
	}
	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return s.issueSession(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil
	}
	return s.redis.RevokeRefreshToken(ctx, claims.UserID, claims.ID)
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
