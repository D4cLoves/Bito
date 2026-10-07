package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

type OTPService struct{}

func NewOTPService() *OTPService {
	return &OTPService{}
}

func (s *OTPService) GenerateNumericOTP() (string, error) {
	maxVal := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, maxVal)
	if err != nil {
		return "", fmt.Errorf("crypto/rand failure: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}
