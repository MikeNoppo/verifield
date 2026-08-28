// Package schema adalah SATU-SATUNYA tempat definisi schema database.
// Padanan schema.prisma — semua tabel, kolom, index, dan relasi ada di file ini.
//
// Aturan main:
//   - Tambah tabel baru  → tambahkan struct di bawah, lalu daftarkan di All().
//   - Ubah kolom         → ubah tag `gorm:"..."` pada struct-nya.
//   - Generate migrasi   → make migrate-diff name=nama_perubahan
//   - Terapkan migrasi   → make migrate-up
//
// Paket ini sengaja tidak mengimpor paket lain milik aplikasi (paket daun),
// sehingga relasi antar tabel lintas modul tidak pernah menimbulkan import cycle.
//
// # Cara menulis relasi
//
// Belongs-to (banyak Product dimiliki satu User) — kolom foreign key ditulis
// eksplisit, dan field relasi memakai tag `gorm:"foreignKey:..."`:
//
//	type Product struct {
//	    ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
//	    OwnerID uuid.UUID `gorm:"type:uuid;not null;index"`
//	    Owner   *User     `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
//	}
//
// Has-many (satu User punya banyak Product) — kebalikannya, ditulis di User:
//
//	Products []Product `gorm:"foreignKey:OwnerID"`
//
// Many-to-many lewat tabel pivot:
//
//	Tags []Tag `gorm:"many2many:product_tags"`
//
// Relasi TIDAK ikut ter-load otomatis saat query. Minta secara eksplisit di
// repository: db.Preload("Owner").Find(&products) — padanan `include` di Prisma.
package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// All adalah daftar seluruh tabel, dibaca oleh cmd/atlas-loader saat Atlas
// menghitung diff migrasi. Struct yang tidak terdaftar di sini tidak akan
// pernah muncul di berkas migrasi.
func All() []any {
	return []any{
		&User{},
	}
}

// ---------------------------------------------------------------------------
// Enum
// ---------------------------------------------------------------------------

// Role membatasi nilai kolom users.role.
type Role string

const (
	// RoleAdmin adalah koordinator operasional / dispatcher. Dokumen konteks
	// bisnis bagian 5.2 menyamakan keduanya, jadi tidak ada role terpisah.
	RoleAdmin     Role = "admin"
	RoleClient    Role = "client"
	RoleInspector Role = "inspector"
	RoleCS        Role = "cs"
)

// ---------------------------------------------------------------------------
// Tabel: users
// ---------------------------------------------------------------------------

// User memetakan tabel `users`.
type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"                     json:"id"`
	Name      string         `gorm:"type:varchar(120);not null"               json:"name"`
	Email     string         `gorm:"type:varchar(160);not null;uniqueIndex"   json:"email"`
	Password  string         `gorm:"type:varchar(255);not null"               json:"-"` // jangan pernah dikirim ke client
	Role      Role           `gorm:"type:varchar(20);not null;default:client" json:"role"`
	IsActive  bool           `gorm:"not null;default:true"                    json:"is_active"`
	CreatedAt time.Time      `                                                json:"created_at"`
	UpdatedAt time.Time      `                                                json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                    json:"-"` // soft delete
}

func (User) TableName() string { return "users" }

// BeforeCreate mengisi primary key UUID kalau belum diset.
func (u *User) BeforeCreate(*gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.Role == "" {
		u.Role = RoleClient
	}
	return nil
}

// IsAdmin dipakai service untuk pengecekan hak akses tingkat domain.
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }
