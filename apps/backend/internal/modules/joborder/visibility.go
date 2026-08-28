package joborder

import (
	"verifield-be/internal/common/apperror"
	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/schema"
)

var errNoCompany = apperror.Forbidden("Akun klien ini belum terhubung ke perusahaan mana pun")

// Viewer adalah identitas yang menentukan batas baca, dinyatakan dalam id
// mentah supaya jalur baca (schema.JobOrder) dan jalur siaran
// (dto.JobOrderResponse) memakai aturan yang sama tanpa saling mengenal bentuk
// data masing-masing.
type Viewer struct {
	Role      schema.Role
	ActorID   string
	CompanyID string
}

// Target adalah order yang sedang dinilai. Field kosong berarti tidak ada:
// InspectorID kosong artinya order belum ditugaskan.
type Target struct {
	CompanyID   string
	InspectorID string
}

// CanSee adalah satu-satunya tempat batas baca per peran dinyatakan.
//
// Klien terbatas pada order perusahaannya (A-03), inspektor hanya pada order
// yang ditugaskan kepadanya. Koordinator dan CS melihat semuanya.
func CanSee(v Viewer, t Target) bool {
	switch v.Role {
	case schema.RoleClient:
		return v.CompanyID != "" && t.CompanyID == v.CompanyID
	case schema.RoleInspector:
		return v.ActorID != "" && t.InspectorID == v.ActorID
	default:
		return true
	}
}

// ViewerOf menerjemahkan aktor permintaan menjadi Viewer.
func ViewerOf(actor Actor) Viewer {
	v := Viewer{Role: actor.Role, ActorID: actor.ID.String()}
	if actor.CompanyID != nil {
		v.CompanyID = actor.CompanyID.String()
	}
	return v
}

// VisibleTo memutuskan apakah aktor berhak melihat sebuah order.
func VisibleTo(actor Actor, order *schema.JobOrder) bool {
	t := Target{CompanyID: order.CompanyID.String()}
	if order.InspectorID != nil {
		t.InspectorID = order.InspectorID.String()
	}
	return CanSee(ViewerOf(actor), t)
}

// VisibleInResponse menilai order yang sudah berbentuk response, untuk jalur
// siaran yang tidak memegang baris database.
func VisibleInResponse(v Viewer, order dto.JobOrderResponse) bool {
	t := Target{CompanyID: order.CompanyID}
	if order.InspectorID != nil {
		t.InspectorID = *order.InspectorID
	}
	return CanSee(v, t)
}

// ScopeQuery mempersempit saringan daftar ke batas baca aktor.
//
// Batas yang sama dengan CanSee, tetapi dinyatakan sebagai saringan query
// karena daftar tidak bisa memuat seluruh order lalu membuang yang tak boleh
// dilihat — halamannya akan bolong. Nilai yang dikirim lewat query string
// ditimpa, bukan dipercaya.
func ScopeQuery(actor Actor, query dto.ListQuery) (dto.ListQuery, error) {
	switch actor.Role {
	case schema.RoleClient:
		if actor.CompanyID == nil {
			return query, errNoCompany
		}
		query.CompanyID = actor.CompanyID.String()
	case schema.RoleInspector:
		query.InspectorID = actor.ID.String()
	}
	return query, nil
}
