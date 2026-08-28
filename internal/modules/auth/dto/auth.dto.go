// Package dto berisi kontrak request/response modul auth.
package dto

import (
	"time"

	userdto "verifield-be/internal/modules/user/dto"
)

// RegisterDTO adalah body endpoint POST /auth/register.
type RegisterDTO struct {
	Name     string `json:"name"     binding:"required,min=3,max=120" example:"Siti Rahma"`
	Email    string `json:"email"    binding:"required,email,max=160" example:"siti@verifield.id"`
	Password string `json:"password" binding:"required,min=8,max=72"  example:"rahasia123"`
}

// LoginDTO adalah body endpoint POST /auth/login.
type LoginDTO struct {
	Email    string `json:"email"    binding:"required,email"    example:"siti@verifield.id"`
	Password string `json:"password" binding:"required,min=8"    example:"rahasia123"`
}

// RefreshTokenDTO adalah body endpoint POST /auth/refresh.
type RefreshTokenDTO struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOi..."`
}

// TokenResponse adalah pasangan token yang diterbitkan setelah login.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type" example:"Bearer"`
	ExpiresIn    int64     `json:"expires_in" example:"900"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthResponse adalah hasil register/login: data user beserta token-nya.
type AuthResponse struct {
	User  userdto.UserResponse `json:"user"`
	Token TokenResponse        `json:"token"`
}
