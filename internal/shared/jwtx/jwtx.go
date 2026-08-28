// Package jwtx menangani pembuatan dan verifikasi JWT. Dipakai bersama oleh
// modul auth (menerbitkan token) dan middleware JWTAuth (memverifikasi token),
// sehingga tidak terjadi import cycle antara middleware dan modul.
package jwtx

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Jenis token yang diterbitkan.
const (
	TypeAccess  = "access"
	TypeRefresh = "refresh"
)

var (
	ErrInvalidToken = errors.New("token tidak valid")
	ErrExpiredToken = errors.New("token sudah kedaluwarsa")
	ErrWrongType    = errors.New("jenis token tidak sesuai")
)

// Claims adalah isi JWT milik aplikasi ini.
type Claims struct {
	jwt.RegisteredClaims
	Role      string `json:"role"`
	TokenType string `json:"typ"`
}

// Manager menerbitkan dan memverifikasi token memakai HMAC-SHA256.
type Manager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (m *Manager) AccessTTL() time.Duration  { return m.accessTTL }
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// GenerateAccess menerbitkan access token untuk user tertentu.
func (m *Manager) GenerateAccess(userID, role string) (string, time.Time, error) {
	return m.generate(userID, role, TypeAccess, m.accessTTL)
}

// GenerateRefresh menerbitkan refresh token untuk user tertentu.
func (m *Manager) GenerateRefresh(userID, role string) (string, time.Time, error) {
	return m.generate(userID, role, TypeRefresh, m.refreshTTL)
}

func (m *Manager) generate(userID, role, tokenType string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Role:      role,
		TokenType: tokenType,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// Parse memverifikasi tanda tangan token dan memastikan jenisnya sesuai.
func (m *Manager) Parse(tokenString, expectedType string) (*Claims, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, ErrWrongType
	}
	if claims.Subject == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
