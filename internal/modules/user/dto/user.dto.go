// Package dto berisi kontrak request/response modul user.
// Padanan *.dto.ts + class-validator di NestJS.
package dto

import (
	"time"

	"verifield-be/internal/schema"
)

// CreateUserDTO adalah body untuk membuat user baru.
type CreateUserDTO struct {
	Name     string `json:"name"     binding:"required,min=3,max=120"  example:"Siti Rahma"`
	Email    string `json:"email"    binding:"required,email,max=160"  example:"siti@verifield.id"`
	Password string `json:"password" binding:"required,min=8,max=72"   example:"rahasia123"`
	Role     string `json:"role"     binding:"omitempty,oneof=admin client inspector cs" example:"client"`
}

// UpdateUserDTO memakai pointer supaya field yang tidak dikirim tidak ikut
// tertimpa — padanan PartialType() di NestJS.
type UpdateUserDTO struct {
	Name     *string `json:"name"      binding:"omitempty,min=3,max=120"     example:"Siti Rahma"`
	Email    *string `json:"email"     binding:"omitempty,email,max=160"     example:"siti@verifield.id"`
	Password *string `json:"password"  binding:"omitempty,min=8,max=72"      example:"rahasia123"`
	Role     *string `json:"role"      binding:"omitempty,oneof=admin client inspector cs"  example:"admin"`
	IsActive *bool   `json:"is_active"                                       example:"true"`
}

// UserResponse adalah bentuk user yang aman dikirim ke client.
type UserResponse struct {
	ID        string    `json:"id"         example:"6f1e6f0c-6f2a-4c5e-9f3a-0b6b1a4d2c11"`
	Name      string    `json:"name"       example:"Siti Rahma"`
	Email     string    `json:"email"      example:"siti@verifield.id"`
	Role      string    `json:"role"       example:"client"`
	IsActive  bool      `json:"is_active"  example:"true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToUserResponse memetakan entity ke response DTO.
func ToUserResponse(u *schema.User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Name:      u.Name,
		Email:     u.Email,
		Role:      string(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// ToUserResponses memetakan slice entity ke slice response DTO.
func ToUserResponses(users []schema.User) []UserResponse {
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, ToUserResponse(&users[i]))
	}
	return out
}
