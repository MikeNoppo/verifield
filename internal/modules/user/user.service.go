package user

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/pagination"
	"verifield-be/internal/common/response"
	"verifield-be/internal/modules/user/dto"
	"verifield-be/internal/schema"
	"verifield-be/internal/shared/hash"
)

// Service berisi business logic modul user. Padanan *.service.ts di NestJS.
type Service interface {
	Create(ctx context.Context, input dto.CreateUserDTO) (*dto.UserResponse, error)
	FindAll(ctx context.Context, query pagination.Query) ([]dto.UserResponse, response.Meta, error)
	FindByID(ctx context.Context, id string) (*dto.UserResponse, error)
	Update(ctx context.Context, id string, input dto.UpdateUserDTO) (*dto.UserResponse, error)
	Remove(ctx context.Context, id string) error

	// Dipakai modul auth — mengembalikan entity (termasuk hash password),
	// jadi jangan pernah dikirim langsung sebagai response HTTP.
	CreateEntity(ctx context.Context, input dto.CreateUserDTO) (*schema.User, error)
	FindEntityByEmail(ctx context.Context, email string) (*schema.User, error)
	FindEntityByID(ctx context.Context, id string) (*schema.User, error)
}

type service struct {
	repo Repository
}

// NewService merakit service dari repository-nya.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, input dto.CreateUserDTO) (*dto.UserResponse, error) {
	user, err := s.CreateEntity(ctx, input)
	if err != nil {
		return nil, err
	}
	res := dto.ToUserResponse(user)
	return &res, nil
}

func (s *service) CreateEntity(ctx context.Context, input dto.CreateUserDTO) (*schema.User, error) {
	email := normalizeEmail(input.Email)

	exists, err := s.repo.ExistsByEmail(ctx, email, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.Conflict("Email sudah terdaftar")
	}

	hashed, err := hash.Password(input.Password)
	if err != nil {
		return nil, apperror.Internal("Gagal memproses password").Wrap(err)
	}

	role := schema.Role(input.Role)
	if role == "" {
		role = schema.RoleUser
	}

	user := &schema.User{
		Name:     strings.TrimSpace(input.Name),
		Email:    email,
		Password: hashed,
		Role:     role,
		IsActive: true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperror.Conflict("Email sudah terdaftar")
		}
		return nil, err
	}

	return user, nil
}

func (s *service) FindAll(ctx context.Context, query pagination.Query) ([]dto.UserResponse, response.Meta, error) {
	query.Normalize(SortableColumns, "created_at")

	users, total, err := s.repo.FindAll(ctx, query)
	if err != nil {
		return nil, response.Meta{}, err
	}

	return dto.ToUserResponses(users), pagination.BuildMeta(query, total), nil
}

func (s *service) FindByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := s.FindEntityByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := dto.ToUserResponse(user)
	return &res, nil
}

func (s *service) FindEntityByID(ctx context.Context, id string) (*schema.User, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(ctx, parsedID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("User tidak ditemukan")
		}
		return nil, err
	}
	return user, nil
}

func (s *service) FindEntityByEmail(ctx context.Context, email string) (*schema.User, error) {
	user, err := s.repo.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("User tidak ditemukan")
		}
		return nil, err
	}
	return user, nil
}

func (s *service) Update(ctx context.Context, id string, input dto.UpdateUserDTO) (*dto.UserResponse, error) {
	user, err := s.FindEntityByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Email != nil {
		email := normalizeEmail(*input.Email)
		exists, err := s.repo.ExistsByEmail(ctx, email, user.ID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, apperror.Conflict("Email sudah dipakai user lain")
		}
		user.Email = email
	}

	if input.Name != nil {
		user.Name = strings.TrimSpace(*input.Name)
	}
	if input.Role != nil {
		user.Role = schema.Role(*input.Role)
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}
	if input.Password != nil {
		hashed, err := hash.Password(*input.Password)
		if err != nil {
			return nil, apperror.Internal("Gagal memproses password").Wrap(err)
		}
		user.Password = hashed
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	res := dto.ToUserResponse(user)
	return &res, nil
}

func (s *service) Remove(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, parsedID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("User tidak ditemukan")
		}
		return err
	}
	return nil
}

func parseID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, apperror.BadRequest("Format id tidak valid").Wrap(err)
	}
	return parsed, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
