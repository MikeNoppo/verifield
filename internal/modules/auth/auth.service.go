package auth

import (
	"context"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/modules/auth/dto"
	"verifield-be/internal/modules/user"
	userdto "verifield-be/internal/modules/user/dto"
	"verifield-be/internal/schema"
	"verifield-be/internal/shared/hash"
)

// Service menangani registrasi, login, dan refresh token.
type Service interface {
	Register(ctx context.Context, input dto.RegisterDTO) (*dto.AuthResponse, error)
	Login(ctx context.Context, input dto.LoginDTO) (*dto.AuthResponse, error)
	Refresh(ctx context.Context, input dto.RefreshTokenDTO) (*dto.TokenResponse, error)
	Profile(ctx context.Context, userID string) (*userdto.UserResponse, error)
}

type service struct {
	users  user.Service // dependency ke modul user — searah, tanpa import cycle
	tokens *TokenService
}

func NewService(users user.Service, tokens *TokenService) Service {
	return &service{users: users, tokens: tokens}
}

func (s *service) Register(ctx context.Context, input dto.RegisterDTO) (*dto.AuthResponse, error) {
	created, err := s.users.CreateEntity(ctx, userdto.CreateUserDTO{
		Name:     input.Name,
		Email:    input.Email,
		Password: input.Password,
		Role:     string(schema.RoleClient), // registrasi publik selalu jadi klien
	})
	if err != nil {
		return nil, err
	}

	return s.buildAuthResponse(created)
}

func (s *service) Login(ctx context.Context, input dto.LoginDTO) (*dto.AuthResponse, error) {
	found, err := s.users.FindEntityByEmail(ctx, input.Email)
	if err != nil {
		// Jangan bedakan "email tidak ada" dan "password salah" — mencegah
		// penyerang memetakan email mana yang terdaftar.
		if _, ok := apperror.As(err); ok {
			return nil, apperror.Unauthorized("Email atau password salah")
		}
		return nil, err
	}

	if !hash.Compare(found.Password, input.Password) {
		return nil, apperror.Unauthorized("Email atau password salah")
	}

	if !found.IsActive {
		return nil, apperror.Forbidden("Akun Anda tidak aktif")
	}

	return s.buildAuthResponse(found)
}

func (s *service) Refresh(ctx context.Context, input dto.RefreshTokenDTO) (*dto.TokenResponse, error) {
	userID, err := s.tokens.ParseRefresh(input.RefreshToken)
	if err != nil {
		return nil, err
	}

	found, err := s.users.FindEntityByID(ctx, userID)
	if err != nil {
		return nil, apperror.Unauthorized("Refresh token tidak valid")
	}

	if !found.IsActive {
		return nil, apperror.Forbidden("Akun Anda tidak aktif")
	}

	token, err := s.tokens.Issue(found)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *service) Profile(ctx context.Context, userID string) (*userdto.UserResponse, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *service) buildAuthResponse(u *schema.User) (*dto.AuthResponse, error) {
	token, err := s.tokens.Issue(u)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		User:  userdto.ToUserResponse(u),
		Token: token,
	}, nil
}
