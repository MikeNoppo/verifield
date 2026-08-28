package user

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"verifield-be/internal/common/pagination"
	"verifield-be/internal/schema"
)

// SortableColumns adalah whitelist kolom yang boleh dipakai pada sort_by.
var SortableColumns = []string{"created_at", "updated_at", "name", "email", "role"}

// Repository adalah kontrak akses data user. Dibuat sebagai interface supaya
// service mudah di-mock saat unit test.
type Repository interface {
	Create(ctx context.Context, user *schema.User) error
	FindAll(ctx context.Context, query pagination.Query) ([]schema.User, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*schema.User, error)
	FindByEmail(ctx context.Context, email string) (*schema.User, error)
	Update(ctx context.Context, user *schema.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string, excludeID uuid.UUID) (bool, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository membuat implementasi Repository berbasis GORM.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, user *schema.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *gormRepository) FindAll(ctx context.Context, query pagination.Query) ([]schema.User, int64, error) {
	tx := r.db.WithContext(ctx).Model(&schema.User{})

	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", pattern, pattern)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []schema.User
	err := tx.Order(query.OrderClause()).
		Limit(query.Limit).
		Offset(query.Offset()).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *gormRepository) FindByID(ctx context.Context, id uuid.UUID) (*schema.User, error) {
	var user schema.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) FindByEmail(ctx context.Context, email string) (*schema.User, error) {
	var user schema.User
	err := r.db.WithContext(ctx).
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) Update(ctx context.Context, user *schema.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *gormRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&schema.User{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *gormRepository) ExistsByEmail(ctx context.Context, email string, excludeID uuid.UUID) (bool, error) {
	tx := r.db.WithContext(ctx).Model(&schema.User{}).
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email)))

	if excludeID != uuid.Nil {
		tx = tx.Where("id <> ?", excludeID)
	}

	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
