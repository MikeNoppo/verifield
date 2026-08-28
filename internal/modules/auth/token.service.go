package auth

import (
	"time"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/modules/auth/dto"
	"verifield-be/internal/schema"
	"verifield-be/internal/shared/jwtx"
)

// TokenService menerbitkan pasangan access + refresh token untuk seorang user.
// Dipisah dari AuthService supaya logika token bisa dipakai ulang dan diuji sendiri.
type TokenService struct {
	manager *jwtx.Manager
}

func NewTokenService(manager *jwtx.Manager) *TokenService {
	return &TokenService{manager: manager}
}

// Issue menerbitkan access token dan refresh token untuk user tertentu.
func (s *TokenService) Issue(user *schema.User) (dto.TokenResponse, error) {
	userID := user.ID.String()
	role := string(user.Role)

	accessToken, expiresAt, err := s.manager.GenerateAccess(userID, role)
	if err != nil {
		return dto.TokenResponse{}, apperror.Internal("Gagal menerbitkan access token").Wrap(err)
	}

	refreshToken, _, err := s.manager.GenerateRefresh(userID, role)
	if err != nil {
		return dto.TokenResponse{}, apperror.Internal("Gagal menerbitkan refresh token").Wrap(err)
	}

	return dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.manager.AccessTTL() / time.Second),
		ExpiresAt:    expiresAt,
	}, nil
}

// ParseRefresh memverifikasi refresh token dan mengembalikan id user pemiliknya.
func (s *TokenService) ParseRefresh(token string) (string, error) {
	claims, err := s.manager.Parse(token, jwtx.TypeRefresh)
	if err != nil {
		return "", apperror.Unauthorized("Refresh token tidak valid atau sudah kedaluwarsa").Wrap(err)
	}
	return claims.Subject, nil
}
