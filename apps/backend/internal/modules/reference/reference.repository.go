package reference

import (
	"context"

	"gorm.io/gorm"

	"verifield-be/internal/schema"
)

// InspectionTypeResponse adalah satu jenis inspeksi yang masih ditawarkan.
type InspectionTypeResponse struct {
	ID   string `json:"id"`
	Code string `json:"code" example:"bulk_cargo"`
	Name string `json:"name" example:"Inspeksi Kargo Curah"`
}

// InspectorResponse adalah satu inspektor beserta beban kerjanya.
//
// ActiveJobs ikut dikirim supaya koordinator bisa memilih berdasarkan beban,
// bukan sekadar berdasarkan nama — penugasan otomatis berada di luar cakupan,
// jadi angka ini yang menggantikannya.
type InspectorResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	ActiveJobs int64  `json:"active_jobs"`
}

// DemoActorResponse adalah satu identitas siap pakai untuk pemilih peran di
// frontend, pengganti login selama autentikasi di luar cakupan.
type DemoActorResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	Role        string  `json:"role" enums:"admin,client,inspector,cs"`
	CompanyID   *string `json:"company_id"`
	CompanyName *string `json:"company_name"`
}

type Repository interface {
	InspectionTypes(ctx context.Context) ([]InspectionTypeResponse, error)
	Inspectors(ctx context.Context) ([]InspectorResponse, error)
	DemoActors(ctx context.Context) ([]DemoActorResponse, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) InspectionTypes(ctx context.Context) ([]InspectionTypeResponse, error) {
	var types []schema.InspectionType
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&types).Error
	if err != nil {
		return nil, err
	}

	out := make([]InspectionTypeResponse, 0, len(types))
	for _, t := range types {
		out = append(out, InspectionTypeResponse{
			ID:   t.ID.String(),
			Code: t.Code,
			Name: t.Name,
		})
	}
	return out, nil
}

func (r *gormRepository) Inspectors(ctx context.Context) ([]InspectorResponse, error) {
	var rows []struct {
		ID         string
		Name       string
		Email      string
		ActiveJobs int64
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.name, u.email,
		       COUNT(jo.id) AS active_jobs
		  FROM users u
		  LEFT JOIN job_orders jo
		         ON jo.inspector_id = u.id
		        AND jo.deleted_at IS NULL
		        AND jo.current_status NOT IN ?
		 WHERE u.role = ? AND u.is_active = true AND u.deleted_at IS NULL
		 GROUP BY u.id, u.name, u.email
		 ORDER BY active_jobs ASC, u.name ASC
	`, []schema.JobStatus{schema.StatusCompleted, schema.StatusFailed, schema.StatusCancelled},
		schema.RoleInspector).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]InspectorResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, InspectorResponse(row))
	}
	return out, nil
}

// DemoActors mengembalikan seluruh pengguna aktif yang bisa dipakai sebagai
// identitas demo, dikelompokkan menurut peran.
//
// Urutan inspektor sengaja bukan alfabetis: inspektor tersibuk ditaruh paling
// depan supaya layar lapangan yang dipilih baku langsung berisi pekerjaan.
// Klien tanpa perusahaan dilewati karena ia tidak bisa melihat order apa pun
// (asumsi A-03).
func (r *gormRepository) DemoActors(ctx context.Context) ([]DemoActorResponse, error) {
	sibuk, err := r.busiestInspector(ctx)
	if err != nil {
		return nil, err
	}

	var users []schema.User
	err = r.db.WithContext(ctx).
		Preload("Company").
		Where("is_active = ?", true).
		Order("role ASC, email ASC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	// Urutan peran mengikuti kemunculannya di hasil query, bukan daftar tetap:
	// peran baru pada skema ikut terbawa tanpa perlu diingat di sini.
	byRole := make(map[schema.Role][]DemoActorResponse, 4)
	urutan := make([]schema.Role, 0, 4)
	for i := range users {
		u := users[i]
		if u.Role == schema.RoleClient && u.CompanyID == nil {
			continue
		}
		if _, sudah := byRole[u.Role]; !sudah {
			urutan = append(urutan, u.Role)
		}
		byRole[u.Role] = append(byRole[u.Role], toActorResponse(u))
	}

	if sibuk != "" {
		for i := range byRole[schema.RoleInspector] {
			if byRole[schema.RoleInspector][i].ID == sibuk {
				ins := byRole[schema.RoleInspector]
				ins[0], ins[i] = ins[i], ins[0]
				break
			}
		}
	}

	out := make([]DemoActorResponse, 0, len(users))
	for _, role := range urutan {
		out = append(out, byRole[role]...)
	}
	return out, nil
}

func toActorResponse(u schema.User) DemoActorResponse {
	actor := DemoActorResponse{
		ID:    u.ID.String(),
		Name:  u.Name,
		Email: u.Email,
		Role:  string(u.Role),
	}
	if u.CompanyID != nil {
		id := u.CompanyID.String()
		actor.CompanyID = &id
	}
	if u.Company != nil {
		name := u.Company.Name
		actor.CompanyName = &name
	}
	return actor
}

// busiestInspector mengembalikan id inspektor dengan penugasan berjalan
// terbanyak, atau string kosong bila tidak ada satu pun yang sedang bertugas.
func (r *gormRepository) busiestInspector(ctx context.Context) (string, error) {
	inspectors, err := r.Inspectors(ctx)
	if err != nil {
		return "", err
	}

	// Inspectors() mengurutkan menaik berdasarkan beban, jadi yang paling sibuk
	// ada di ujung.
	for i := len(inspectors) - 1; i >= 0; i-- {
		if inspectors[i].ActiveJobs > 0 {
			return inspectors[i].ID, nil
		}
	}
	return "", nil
}
