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
//
// # Jebakan default pada kolom bool
//
// GORM membuang nilai zero dari INSERT untuk setiap field yang tag-nya memuat
// `default:`. Akibatnya `IsActive: false` pada struct bertag `default:true`
// tersimpan sebagai true, tanpa error dan tanpa peringatan. Seluruh jalur
// pembuatan yang ada sekarang mengisi IsActive eksplisit dengan true sehingga
// aman — tetapi begitu ada kebutuhan membuat baris non-aktif, hapus dulu
// `default:true` dari tag-nya. Jangan mengandalkan nilai yang disetel di struct.
package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// All adalah daftar seluruh tabel, dibaca oleh cmd/atlas-loader saat Atlas
// menghitung diff migrasi. Struct yang tidak terdaftar di sini tidak akan
// pernah muncul di berkas migrasi.
//
// Tabel induk didaftarkan lebih dulu supaya urutan foreign key di DDL wajar.
func All() []any {
	return []any{
		&Company{},
		&User{},
		&InspectionType{},
		&JobOrder{},
		&JobStatusEvent{},
		&CancellationRequest{},
		&JobOrderAlert{},
		&ReferenceCounter{},
	}
}

func ensureUUID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
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

type JobStatus string

const (
	StatusRequested  JobStatus = "requested"
	StatusAssigned   JobStatus = "assigned"
	StatusOnTheWay   JobStatus = "on_the_way"
	StatusOnSite     JobStatus = "on_site"
	StatusInProgress JobStatus = "in_progress"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

func (s JobStatus) IsFinal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled
}

type CancellationStatus string

const (
	CancellationPending  CancellationStatus = "pending"
	CancellationApproved CancellationStatus = "approved"
	CancellationRejected CancellationStatus = "rejected"
	// CancellationPendingSettlement: pekerjaan mencapai status final sebelum
	// koordinator sempat memutuskan. Pembatalannya tidak lagi mungkin, tetapi
	// pertanyaan komersialnya justru baru terbuka — permintaan tetap menunggu
	// keputusan, hanya pertanyaannya yang berganti (keputusan B-10).
	CancellationPendingSettlement CancellationStatus = "pending_settlement"
	CancellationSettled           CancellationStatus = "settled"
)

// SettlementOutcome adalah hasil penyelesaian komersial ketika pekerjaan
// terlanjur selesai mendahului keputusan pembatalan.
//
// Permintaan klien tidak cacat: ia masuk selagi pekerjaan masih berjalan, dan
// yang membuat pekerjaan tetap jalan adalah keputusan perusahaan sendiri
// (B-05). Karena itu klien punya klaim yang sah, dan hasilnya tidak bisa
// diasumsikan selalu "tagih penuh".
type SettlementOutcome string

const (
	SettlementBilledFull    SettlementOutcome = "billed_full"
	SettlementBilledPartial SettlementOutcome = "billed_partial"
	SettlementWaived        SettlementOutcome = "waived"
)

type AlertType string

const (
	AlertLateUpdateRejected AlertType = "late_update_rejected"
)

// ---------------------------------------------------------------------------
// Tabel: companies
// ---------------------------------------------------------------------------

// Company memetakan tabel `companies` — perusahaan klien pemesan jasa.
//
// Kepemilikan job order melekat ke perusahaan, bukan ke user, karena beberapa
// user klien dari perusahaan yang sama harus melihat daftar order yang sama
// (asumsi A-03).
type Company struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"                     json:"id"`
	Code      string         `gorm:"type:varchar(20);not null;uniqueIndex"    json:"code"`
	Name      string         `gorm:"type:varchar(160);not null"               json:"name"`
	IsActive  bool           `gorm:"not null;default:true"                    json:"is_active"`
	CreatedAt time.Time      `                                                json:"created_at"`
	UpdatedAt time.Time      `                                                json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                    json:"-"`
}

func (Company) TableName() string { return "companies" }

func (c *Company) BeforeCreate(*gorm.DB) error {
	ensureUUID(&c.ID)
	return nil
}

// ---------------------------------------------------------------------------
// Tabel: users
// ---------------------------------------------------------------------------

// User memetakan tabel `users`.
type User struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"                       json:"id"`
	Name     string    `gorm:"type:varchar(120);not null"                 json:"name"`
	Email    string    `gorm:"type:varchar(160);not null;uniqueIndex"     json:"email"`
	Password string    `gorm:"type:varchar(255);not null"                 json:"-"` // jangan pernah dikirim ke client
	Role     Role      `gorm:"type:varchar(20);not null;default:client"   json:"role"`
	IsActive bool      `gorm:"not null;default:true"                      json:"is_active"`

	// Hanya terisi untuk role client; staf internal tidak terikat perusahaan mana pun.
	CompanyID *uuid.UUID `gorm:"type:uuid;index"       json:"company_id,omitempty"`
	Company   *Company   `gorm:"foreignKey:CompanyID"  json:"company,omitempty"`

	CreatedAt time.Time      `                json:"created_at"`
	UpdatedAt time.Time      `                json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"    json:"-"`
}

func (User) TableName() string { return "users" }
func (u *User) BeforeCreate(*gorm.DB) error {
	ensureUUID(&u.ID)
	if u.Role == "" {
		u.Role = RoleClient
	}
	return nil
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// ---------------------------------------------------------------------------
// Tabel: inspection_types
// ---------------------------------------------------------------------------

// InspectionType memetakan tabel `inspection_types` — daftar jenis inspeksi.
//
// Sengaja tanpa soft delete: baris ini dirujuk foreign key oleh job order
// historis dan tidak boleh hilang. Jenis yang tidak ditawarkan lagi cukup
// di-nonaktifkan lewat IsActive.
type InspectionType struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"                     json:"id"`
	Code      string    `gorm:"type:varchar(40);not null;uniqueIndex"    json:"code"`
	Name      string    `gorm:"type:varchar(120);not null"               json:"name"`
	IsActive  bool      `gorm:"not null;default:true"                    json:"is_active"`
	CreatedAt time.Time `                                                json:"created_at"`
	UpdatedAt time.Time `                                                json:"updated_at"`
}

func (InspectionType) TableName() string { return "inspection_types" }

func (t *InspectionType) BeforeCreate(*gorm.DB) error {
	ensureUUID(&t.ID)
	return nil
}

// ---------------------------------------------------------------------------
// Tabel: job_orders
// ---------------------------------------------------------------------------

// JobOrder memetakan tabel `job_orders` — satu permintaan pemeriksaan, atas
// satu objek, di satu lokasi, pada satu rentang waktu (asumsi A-01 dan A-02).
type JobOrder struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"                     json:"id"`
	ReferenceNumber string    `gorm:"type:varchar(30);not null;uniqueIndex"    json:"reference_number"`

	CompanyID uuid.UUID `gorm:"type:uuid;not null;index:idx_job_orders_company_created,priority:1"   json:"company_id"`
	Company   *Company  `gorm:"foreignKey:CompanyID"                                                 json:"company,omitempty"`

	CreatedByID uuid.UUID `gorm:"type:uuid;not null;index"    json:"created_by_id"`
	CreatedBy   *User     `gorm:"foreignKey:CreatedByID"      json:"created_by,omitempty"`

	InspectionTypeID uuid.UUID       `gorm:"type:uuid;not null;index"       json:"inspection_type_id"`
	InspectionType   *InspectionType `gorm:"foreignKey:InspectionTypeID"    json:"inspection_type,omitempty"`

	ObjectDescription string `gorm:"type:varchar(255);not null"    json:"object_description"`
	LocationName      string `gorm:"type:varchar(160);not null"    json:"location_name"`
	LocationAddress   string `gorm:"type:text;not null"            json:"location_address"`
	City              string `gorm:"type:varchar(80);not null"     json:"city"`

	ScheduledStartAt time.Time `gorm:"not null"    json:"scheduled_start_at"`
	ScheduledEndAt   time.Time `gorm:"not null"    json:"scheduled_end_at"`

	// Kosong selama status masih Requested — inspektor belum ditentukan.
	InspectorID *uuid.UUID `gorm:"type:uuid;index:idx_job_orders_inspector_status,priority:1"    json:"inspector_id,omitempty"`
	Inspector   *User      `gorm:"foreignKey:InspectorID"                                        json:"inspector,omitempty"`

	// WARNING: kedua kolom ini cache baca-cepat, BUKAN sumber kebenaran
	// (keputusan B-01) — sumbernya job_status_events. Hanya boleh ditulis di
	// transaksi yang sama dengan penyisipan event ber-Accepted=true, dan selalu
	// bisa dibangun ulang dari event Accepted=true dengan Seq tertinggi.
	CurrentStatus   JobStatus `gorm:"type:varchar(20);not null;default:requested;index:idx_job_orders_inspector_status,priority:2;index:idx_job_orders_status_changed,priority:1"   json:"current_status"`
	StatusChangedAt time.Time `gorm:"not null;index:idx_job_orders_status_changed,priority:2"                                                                                       json:"status_changed_at"`

	// Optimistic locking (keputusan B-09): UPDATE wajib menyertakan
	// WHERE version = ? supaya perubahan pertama yang menang.
	Version int `gorm:"not null;default:1"    json:"version"`

	CreatedAt time.Time      `gorm:"index:idx_job_orders_company_created,priority:2,sort:desc"    json:"created_at"`
	UpdatedAt time.Time      `                                                                    json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                                        json:"-"`

	// NOTE: constraint ditulis di sisi has-many ini, bukan di sisi belongs-to
	// JobStatusEvent.JobOrder — GORM memakai definisi has-many dan mengabaikan
	// yang sebelah sana, sehingga CASCADE hilang kalau hanya ditulis di situ.
	StatusEvents []JobStatusEvent `gorm:"foreignKey:JobOrderID;constraint:OnDelete:CASCADE"    json:"status_events,omitempty"`
}

func (JobOrder) TableName() string { return "job_orders" }

func (j *JobOrder) BeforeCreate(*gorm.DB) error {
	ensureUUID(&j.ID)
	if j.CurrentStatus == "" {
		j.CurrentStatus = StatusRequested
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tabel: job_status_events
// ---------------------------------------------------------------------------

// JobStatusEvent memetakan tabel `job_status_events` — riwayat setiap kejadian
// perubahan status.
//
// WARNING: tabel ini bersifat MENAMBAH SAJA (keputusan B-01). Baris di sini
// tidak pernah di-update maupun dihapus, karena itulah struct ini sengaja tidak
// punya UpdatedAt maupun DeletedAt. Koreksi status ditulis sebagai baris baru
// ber-IsCorrection=true, bukan dengan mengubah baris lama.
type JobStatusEvent struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"    json:"id"`

	// Kursor monotonik urutan penerimaan server. Dipakai untuk menentukan status
	// terkini dan untuk mengirim ulang event yang terlewat saat klien reconnect.
	//
	// WARNING: nilainya diisi database (bigserial), bukan Go. GORM hanya
	// mengembalikan primary key setelah INSERT, jadi sertakan clause.Returning
	// untuk kolom ini bila Seq dibutuhkan langsung setelah Create.
	Seq int64 `gorm:"autoIncrement;uniqueIndex"    json:"seq"`

	JobOrderID uuid.UUID `gorm:"type:uuid;not null;index:idx_events_order_occurred,priority:1;uniqueIndex:idx_events_idempotency,priority:1"    json:"job_order_id"`
	JobOrder   *JobOrder `gorm:"foreignKey:JobOrderID;constraint:OnDelete:CASCADE"                                                             json:"-"`

	FromStatus *JobStatus `gorm:"type:varchar(20)"             json:"from_status,omitempty"` // kosong pada event pertama
	ToStatus   JobStatus  `gorm:"type:varchar(20);not null"    json:"to_status"`

	ActorID   *uuid.UUID `gorm:"type:uuid;index"              json:"actor_id,omitempty"` // kosong bila dibuat sistem
	Actor     *User      `gorm:"foreignKey:ActorID"           json:"actor,omitempty"`
	ActorRole Role       `gorm:"type:varchar(20);not null"    json:"actor_role"`

	// Keputusan B-02: waktu kejadian di lapangan dan waktu terima server dicatat
	// terpisah. Timeline klien diurutkan dengan OccurredAt, sedangkan status
	// terkini ditentukan dengan Seq.
	OccurredAt time.Time `gorm:"not null;index:idx_events_order_occurred,priority:2"    json:"occurred_at"`
	ReceivedAt time.Time `gorm:"not null"                                               json:"received_at"`
	// Jam perangkat inspektor di luar batas wajar; OccurredAt jatuh ke ReceivedAt.
	OccurredAtAdjusted bool `gorm:"not null;default:false"    json:"occurred_at_adjusted"`

	// Keputusan B-03: penanda unik yang dibuat perangkat saat offline. Kosong
	// untuk event yang dibuat sistem — di Postgres NULL tidak pernah bertabrakan
	// di unique index, jadi event sistem bebas berulang.
	ClientEventID *string `gorm:"type:varchar(64);uniqueIndex:idx_events_idempotency,priority:2"    json:"client_event_id,omitempty"`

	// Keputusan B-07: event yang datang setelah status final ditolak, tetapi
	// tetap dicatat dengan Accepted=false supaya pekerjaan nyata inspektor tidak
	// hilang begitu saja. Hanya event Accepted=true yang mengubah status.
	//
	// WARNING: kolom ini sengaja TIDAK diberi default database. GORM membuang
	// nilai zero dari INSERT untuk field yang tag-nya memuat `default:`, jadi
	// `Accepted: false` akan tersimpan sebagai true tanpa error — persis
	// mematikan keputusan B-07. Setiap penyisipan wajib mengisinya eksplisit.
	Accepted        bool    `gorm:"not null"           json:"accepted"`
	RejectionReason *string `gorm:"type:varchar(40)"   json:"rejection_reason,omitempty"`

	// Keputusan B-06: koreksi mundur adalah wewenang koordinator dan wajib beralasan.
	IsCorrection bool    `gorm:"not null;default:false"    json:"is_correction"`
	Reason       *string `gorm:"type:text"                 json:"reason,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

func (JobStatusEvent) TableName() string { return "job_status_events" }

func (e *JobStatusEvent) BeforeCreate(*gorm.DB) error {
	ensureUUID(&e.ID)
	return nil
}

// ---------------------------------------------------------------------------
// Tabel: cancellation_requests
// ---------------------------------------------------------------------------

// CancellationRequest memetakan tabel `cancellation_requests`.
//
// Keputusan B-05: pembatalan yang diajukan saat pekerjaan sudah In Progress
// bukan tindakan langsung, melainkan permintaan yang menunggu keputusan
// koordinator.
//
// NOTE: aturan "hanya boleh ada satu permintaan pending per order" ditegakkan di
// service, bukan lewat partial unique index. Index parsial tidak bisa ditulis
// lewat tag GORM, dan menambahkannya dengan tangan ke berkas migrasi akan
// dianggap drift oleh Atlas lalu di-DROP pada diff berikutnya.
type CancellationRequest struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"                                           json:"id"`
	JobOrderID uuid.UUID `gorm:"type:uuid;not null;index:idx_cancel_order_status,priority:1"    json:"job_order_id"`
	JobOrder   *JobOrder `gorm:"foreignKey:JobOrderID;constraint:OnDelete:CASCADE"              json:"-"`

	RequestedByID uuid.UUID `gorm:"type:uuid;not null"         json:"requested_by_id"`
	RequestedBy   *User     `gorm:"foreignKey:RequestedByID"   json:"requested_by,omitempty"`
	Reason        string    `gorm:"type:text;not null"         json:"reason"`

	Status CancellationStatus `gorm:"type:varchar(20);not null;default:pending;index:idx_cancel_order_status,priority:2"    json:"status"`

	DecidedByID  *uuid.UUID `gorm:"type:uuid"                 json:"decided_by_id,omitempty"`
	DecidedBy    *User      `gorm:"foreignKey:DecidedByID"    json:"decided_by,omitempty"`
	DecidedAt    *time.Time `                                 json:"decided_at,omitempty"`
	DecisionNote *string    `gorm:"type:text"                 json:"decision_note,omitempty"`

	// Terisi hanya pada status settled. Dipisahkan dari DecisionNote karena
	// yang satu dapat dihitung untuk laporan, yang lain hanya dapat dibaca.
	SettlementOutcome *SettlementOutcome `gorm:"type:varchar(20)"    json:"settlement_outcome,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CancellationRequest) TableName() string { return "cancellation_requests" }

func (r *CancellationRequest) BeforeCreate(*gorm.DB) error {
	ensureUUID(&r.ID)
	if r.Status == "" {
		r.Status = CancellationPending
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tabel: job_order_alerts
// ---------------------------------------------------------------------------

// JobOrderAlert memetakan tabel `job_order_alerts` — tanda bagi koordinator
// yang perlu ditindaklanjuti manusia.
//
// Ada sebagai tabel tersendiri karena keputusan B-07 menuntut tanda yang bisa
// DISELESAIKAN (ResolvedAt), sedangkan job_status_events bersifat menambah saja
// sehingga tidak boleh punya kolom yang berubah.
//
// NOTE: keterlambatan pembaruan 8 jam sengaja tidak disimpan di sini. Kondisi
// itu murni turunan dari CurrentStatus yang belum final dan StatusChangedAt yang
// sudah lewat, jadi cukup dihitung lewat query dan hilang sendiri begitu
// inspektor memperbarui status.
type JobOrderAlert struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"                                  json:"id"`
	JobOrderID uuid.UUID `gorm:"type:uuid;not null;index"                              json:"job_order_id"`
	JobOrder   *JobOrder `gorm:"foreignKey:JobOrderID;constraint:OnDelete:CASCADE"     json:"-"`

	Type          AlertType       `gorm:"type:varchar(30);not null"                             json:"type"`
	SourceEventID *uuid.UUID      `gorm:"type:uuid"                                             json:"source_event_id,omitempty"`
	SourceEvent   *JobStatusEvent `gorm:"foreignKey:SourceEventID;constraint:OnDelete:CASCADE"  json:"source_event,omitempty"`
	Message       string          `gorm:"type:text;not null"                                    json:"message"`

	ResolvedAt   *time.Time `gorm:"index"       json:"resolved_at,omitempty"`
	ResolvedByID *uuid.UUID `gorm:"type:uuid"   json:"resolved_by_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

func (JobOrderAlert) TableName() string { return "job_order_alerts" }

func (a *JobOrderAlert) BeforeCreate(*gorm.DB) error {
	ensureUUID(&a.ID)
	return nil
}

// ---------------------------------------------------------------------------
// Tabel: reference_counters
// ---------------------------------------------------------------------------

// ReferenceCounter memetakan tabel `reference_counters` — sumber urut nomor
// referensi yang dibaca manusia, misalnya JO-2026-0001.
//
// Dipakai lewat SELECT ... FOR UPDATE di transaksi yang sama dengan penyisipan
// job order. COUNT(*)+1 tidak dipakai karena rawan balapan, dan sequence
// Postgres tidak bisa lahir dari struct GORM.
type ReferenceCounter struct {
	Scope      string    `gorm:"type:varchar(20);primaryKey"    json:"scope"`
	Year       int       `gorm:"primaryKey"                     json:"year"`
	LastNumber int64     `gorm:"not null;default:0"             json:"last_number"`
	UpdatedAt  time.Time `                                      json:"updated_at"`
}

func (ReferenceCounter) TableName() string { return "reference_counters" }
