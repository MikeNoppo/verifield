package joborder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/schema"
)

// SortableColumns adalah whitelist kolom yang boleh dipakai pada sort_by.
var SortableColumns = []string{
	"created_at", "updated_at", "status_changed_at",
	"scheduled_start_at", "reference_number", "current_status",
}

// staleAfter adalah ambang "tanpa pembaruan" pada layar koordinator. Kondisi ini
// sengaja dihitung lewat query, bukan disimpan sebagai baris alert, supaya ia
// hilang dengan sendirinya begitu inspektor memperbarui status.
const staleAfter = 8 * time.Hour

// Repository adalah kontrak akses data job order.
//
// Transaction membungkus satu unit kerja: seluruh metode tulis di bawahnya
// dipanggil lewat Repository hasil callback, yang sudah terikat pada transaksi
// yang sama. Dengan begitu service tidak pernah menyentuh *gorm.DB.
type Repository interface {
	Transaction(ctx context.Context, fn func(tx Repository) error) error

	FindAll(ctx context.Context, query dto.ListQuery) ([]schema.JobOrder, map[uuid.UUID]dto.Derived, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*schema.JobOrder, dto.Derived, error)

	// FindByIDCompact sama dengan FindByID tetapi tanpa riwayat status.
	// Dipakai jalur real-time, yang mengirim satu pesan ke setiap klien pada
	// setiap perubahan — menyertakan riwayat di sana membuat ukuran pesan
	// tumbuh seiring umur order, dan seluruh riwayat dikirim ulang berkali-kali
	// hanya karena satu entri bertambah.
	FindByIDCompact(ctx context.Context, id uuid.UUID) (*schema.JobOrder, dto.Derived, error)
	FindEvents(ctx context.Context, orderID uuid.UUID, afterSeq int64) ([]schema.JobStatusEvent, error)

	// OrderIDsChangedSince mengembalikan id order yang punya event lebih baru
	// dari cursor, terurut menaik. Dipakai untuk memutar ulang perubahan yang
	// terlewat saat klien menyambung kembali.
	OrderIDsChangedSince(ctx context.Context, seq int64) ([]uuid.UUID, error)

	// LockOrder mengambil order dengan SELECT ... FOR UPDATE. Seluruh penulisan
	// event untuk satu order diserialkan lewat kunci ini.
	LockOrder(ctx context.Context, id uuid.UUID) (*schema.JobOrder, error)

	FindEventByClientID(ctx context.Context, orderID uuid.UUID, clientEventID string) (*schema.JobStatusEvent, error)
	InsertEvent(ctx context.Context, event *schema.JobStatusEvent) error
	UpdateOrderStatus(ctx context.Context, order *schema.JobOrder, expectedVersion int) error
	UpdateInspector(ctx context.Context, orderID uuid.UUID, inspectorID uuid.UUID) error

	CreateOrder(ctx context.Context, order *schema.JobOrder) error
	NextReference(ctx context.Context, year int) (string, error)

	InsertAlert(ctx context.Context, alert *schema.JobOrderAlert) error

	// Notify menyiarkan perubahan ke seluruh instance lewat Postgres.
	//
	// Dipanggil di dalam transaksi yang sama dengan penulisan event, karena
	// NOTIFY bersifat transaksional: pesannya baru terkirim saat commit. Dengan
	// begitu mustahil ada pesan real-time untuk perubahan yang ternyata
	// di-rollback — jaminan yang tidak didapat kalau penyiaran dilakukan
	// setelah transaksi selesai.
	Notify(ctx context.Context, seq int64, orderID uuid.UUID) error

	FindPendingCancellation(ctx context.Context, orderID uuid.UUID) (*schema.CancellationRequest, error)
	FindCancellationByID(ctx context.Context, id uuid.UUID) (*schema.CancellationRequest, error)
	CreateCancellation(ctx context.Context, request *schema.CancellationRequest) error
	UpdateCancellation(ctx context.Context, request *schema.CancellationRequest) error
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository membuat implementasi Repository berbasis GORM.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Transaction(ctx context.Context, fn func(tx Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormRepository{db: tx})
	})
}

// ---------------------------------------------------------------------------
// Baca
// ---------------------------------------------------------------------------

// withRelations memuat relasi yang selalu dibutuhkan response. GORM tidak
// pernah memuat relasi otomatis — padanan `include` di Prisma.
func withRelations(tx *gorm.DB) *gorm.DB {
	return tx.
		Preload("Company").
		Preload("CreatedBy").
		Preload("InspectionType").
		Preload("Inspector")
}

func (r *gormRepository) FindAll(ctx context.Context, query dto.ListQuery) ([]schema.JobOrder, map[uuid.UUID]dto.Derived, int64, error) {
	tx := r.db.WithContext(ctx).Model(&schema.JobOrder{})
	tx = applyFilters(tx, query)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, nil, 0, err
	}

	var orders []schema.JobOrder
	err := withRelations(tx).
		Order(query.OrderClause()).
		Limit(query.Limit).
		Offset(query.Offset()).
		Find(&orders).Error
	if err != nil {
		return nil, nil, 0, err
	}

	derived, err := r.derivedFor(ctx, orderIDs(orders))
	if err != nil {
		return nil, nil, 0, err
	}

	return orders, derived, total, nil
}

func applyFilters(tx *gorm.DB, query dto.ListQuery) *gorm.DB {
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		tx = tx.Where(
			"LOWER(reference_number) LIKE ? OR LOWER(object_description) LIKE ? OR LOWER(location_name) LIKE ? OR LOWER(city) LIKE ?",
			pattern, pattern, pattern, pattern,
		)
	}
	if query.Status != "" {
		tx = tx.Where("current_status = ?", query.Status)
	}
	if query.CompanyID != "" {
		tx = tx.Where("company_id = ?", query.CompanyID)
	}
	if query.InspectorID != "" {
		tx = tx.Where("inspector_id = ?", query.InspectorID)
	}

	switch query.Attention {
	case "penugasan":
		tx = tx.Where("current_status = ?", schema.StatusRequested)
	case "pembatalan":
		tx = tx.Where(
			"EXISTS (SELECT 1 FROM cancellation_requests c WHERE c.job_order_id = job_orders.id AND c.status = ?)",
			schema.CancellationPending,
		)
	case "basi":
		tx = tx.Where(
			"current_status NOT IN ? AND status_changed_at < ?",
			finalStatuses(), time.Now().Add(-staleAfter),
		)
	case "terlambat":
		tx = tx.Where(
			"EXISTS (SELECT 1 FROM job_order_alerts a WHERE a.job_order_id = job_orders.id AND a.resolved_at IS NULL)",
		)
	}

	return tx
}

func finalStatuses() []schema.JobStatus {
	return []schema.JobStatus{schema.StatusCompleted, schema.StatusFailed, schema.StatusCancelled}
}

func orderIDs(orders []schema.JobOrder) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(orders))
	for i := range orders {
		ids = append(ids, orders[i].ID)
	}
	return ids
}

// derivedFor menghitung kolom turunan untuk sekumpulan order dalam satu query,
// bukan satu query per order.
//
// NOTE: ketiganya dihitung saat dibaca, tidak didenormalisasi. Pada skala yang
// diasumsikan — puluhan order aktif, bukan puluhan ribu (asumsi A-06) — ini
// jauh lebih murah daripada menjaga tiga kolom turunan tetap konsisten di
// setiap jalur tulis.
func (r *gormRepository) derivedFor(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]dto.Derived, error) {
	result := make(map[uuid.UUID]dto.Derived, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var rows []struct {
		ID                    uuid.UUID
		Seq                   int64
		CancellationRequested bool
		HasOpenAlert          bool
		ExitStatus            *string
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT jo.id,
		       COALESCE((SELECT MAX(e.seq) FROM job_status_events e
		                  WHERE e.job_order_id = jo.id), 0) AS seq,
		       EXISTS (SELECT 1 FROM cancellation_requests c
		                WHERE c.job_order_id = jo.id AND c.status = ?) AS cancellation_requested,
		       EXISTS (SELECT 1 FROM job_order_alerts a
		                WHERE a.job_order_id = jo.id AND a.resolved_at IS NULL) AS has_open_alert,
		       (SELECT e.to_status FROM job_status_events e
		         WHERE e.job_order_id = jo.id AND e.accepted = true
		           AND e.to_status NOT IN ?
		         ORDER BY e.seq DESC LIMIT 1) AS exit_status
		  FROM job_orders jo
		 WHERE jo.id IN ?
	`, schema.CancellationPending, finalStatuses(), ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.ID] = dto.Derived{
			Seq:                   row.Seq,
			CancellationRequested: row.CancellationRequested,
			HasOpenAlert:          row.HasOpenAlert,
			ExitStatus:            row.ExitStatus,
		}
	}
	return result, nil
}

func (r *gormRepository) FindByID(ctx context.Context, id uuid.UUID) (*schema.JobOrder, dto.Derived, error) {
	var order schema.JobOrder

	err := withRelations(r.db.WithContext(ctx)).
		// Riwayat diurutkan berdasarkan waktu kejadian di lapangan, bukan waktu
		// terima — itulah urutan yang benar bagi pembacanya (keputusan B-02).
		// Seq menjadi pemecah seri agar urutannya tetap stabil.
		Preload("StatusEvents", func(tx *gorm.DB) *gorm.DB {
			return tx.Preload("Actor").Order("occurred_at ASC, seq ASC")
		}).
		First(&order, "id = ?", id).Error
	if err != nil {
		return nil, dto.Derived{}, err
	}

	result := derivedForOne(r, ctx, order.ID)
	if result.err != nil {
		return nil, dto.Derived{}, result.err
	}

	return &order, result.derived, nil
}

func (r *gormRepository) FindByIDCompact(ctx context.Context, id uuid.UUID) (*schema.JobOrder, dto.Derived, error) {
	var order schema.JobOrder

	if err := withRelations(r.db.WithContext(ctx)).First(&order, "id = ?", id).Error; err != nil {
		return nil, dto.Derived{}, err
	}

	result := derivedForOne(r, ctx, order.ID)
	if result.err != nil {
		return nil, dto.Derived{}, result.err
	}

	return &order, result.derived, nil
}

type derivedResult struct {
	derived dto.Derived
	err     error
}

// derivedForOne melengkapi kolom turunan dengan detail permintaan pembatalan
// yang tertunda. Sengaja hanya pada endpoint detail: daftar cukup tahu ada atau
// tidaknya, sedangkan koordinator yang membuka satu order butuh id dan alasannya
// untuk bisa memutuskan.
func derivedForOne(r *gormRepository, ctx context.Context, id uuid.UUID) derivedResult {
	byID, err := r.derivedFor(ctx, []uuid.UUID{id})
	if err != nil {
		return derivedResult{err: err}
	}
	derived := byID[id]

	if !derived.CancellationRequested {
		return derivedResult{derived: derived}
	}

	var request schema.CancellationRequest
	err = r.db.WithContext(ctx).
		Preload("RequestedBy").
		Where("job_order_id = ? AND status = ?", id, schema.CancellationPending).
		First(&request).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return derivedResult{derived: derived}
		}
		return derivedResult{err: err}
	}

	pending := &dto.PendingCancellation{
		ID:        request.ID.String(),
		Reason:    request.Reason,
		CreatedAt: request.CreatedAt,
	}
	if request.RequestedBy != nil {
		pending.RequestedByName = request.RequestedBy.Name
	}
	derived.PendingCancellation = pending

	return derivedResult{derived: derived}
}

func (r *gormRepository) FindEvents(ctx context.Context, orderID uuid.UUID, afterSeq int64) ([]schema.JobStatusEvent, error) {
	var events []schema.JobStatusEvent
	err := r.db.WithContext(ctx).
		Preload("Actor").
		Where("job_order_id = ? AND seq > ?", orderID, afterSeq).
		Order("occurred_at ASC, seq ASC").
		Find(&events).Error
	return events, err
}

func (r *gormRepository) OrderIDsChangedSince(ctx context.Context, seq int64) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&schema.JobStatusEvent{}).
		Distinct("job_order_id").
		Where("seq > ?", seq).
		Order("job_order_id").
		Pluck("job_order_id", &ids).Error
	return ids, err
}

// ---------------------------------------------------------------------------
// Tulis
// ---------------------------------------------------------------------------

func (r *gormRepository) LockOrder(ctx context.Context, id uuid.UUID) (*schema.JobOrder, error) {
	var order schema.JobOrder
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&order, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *gormRepository) FindEventByClientID(ctx context.Context, orderID uuid.UUID, clientEventID string) (*schema.JobStatusEvent, error) {
	var event schema.JobStatusEvent
	err := r.db.WithContext(ctx).
		Preload("Actor").
		Where("job_order_id = ? AND client_event_id = ?", orderID, clientEventID).
		First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *gormRepository) InsertEvent(ctx context.Context, event *schema.JobStatusEvent) error {
	// WARNING: seq diisi database (bigserial), bukan Go. Tanpa clause.Returning
	// GORM hanya mengembalikan primary key, dan seq tetap nol — padahal nilainya
	// dibutuhkan langsung sebagai id pesan real-time.
	return r.db.WithContext(ctx).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "seq"}}}).
		Create(event).Error
}

func (r *gormRepository) UpdateOrderStatus(ctx context.Context, order *schema.JobOrder, expectedVersion int) error {
	result := r.db.WithContext(ctx).
		Model(&schema.JobOrder{}).
		// Predikat versi menegakkan invarian keputusan B-09 di lapisan SQL.
		// Pemanggil sudah memegang FOR UPDATE atas baris ini, jadi predikat ini
		// tidak akan pernah meleset — ia ada supaya invariannya terbaca di
		// tempat perubahan benar-benar terjadi.
		Where("id = ? AND version = ?", order.ID, expectedVersion).
		Updates(map[string]any{
			"current_status":    order.CurrentStatus,
			"status_changed_at": order.StatusChangedAt,
			"version":           expectedVersion + 1,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	order.Version = expectedVersion + 1
	return nil
}

func (r *gormRepository) UpdateInspector(ctx context.Context, orderID uuid.UUID, inspectorID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&schema.JobOrder{}).
		Where("id = ?", orderID).
		Update("inspector_id", inspectorID).Error
}

func (r *gormRepository) CreateOrder(ctx context.Context, order *schema.JobOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// NextReference mengalokasikan nomor urut berikutnya dalam satu pernyataan
// atomik. COUNT(*)+1 tidak dipakai karena dua transaksi yang berjalan bersamaan
// akan mendapat nomor yang sama.
func (r *gormRepository) NextReference(ctx context.Context, year int) (string, error) {
	var number int64
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO reference_counters (scope, year, last_number, updated_at)
		VALUES (?, ?, 1, now())
		ON CONFLICT (scope, year)
		DO UPDATE SET last_number = reference_counters.last_number + 1, updated_at = now()
		RETURNING last_number
	`, refScopeJobOrder, year).Scan(&number).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("JO-%d-%04d", year, number), nil
}

const refScopeJobOrder = "job_order"

// NotifyChannel adalah kanal Postgres tempat seluruh perubahan job order
// disiarkan. Listener di modul realtime berlangganan kanal ini.
const NotifyChannel = "verifield_events"

func (r *gormRepository) Notify(ctx context.Context, seq int64, orderID uuid.UUID) error {
	payload := fmt.Sprintf("%d:%s", seq, orderID)
	return r.db.WithContext(ctx).
		Exec("SELECT pg_notify(?, ?)", NotifyChannel, payload).Error
}

func (r *gormRepository) InsertAlert(ctx context.Context, alert *schema.JobOrderAlert) error {
	return r.db.WithContext(ctx).Create(alert).Error
}

func (r *gormRepository) FindPendingCancellation(ctx context.Context, orderID uuid.UUID) (*schema.CancellationRequest, error) {
	var request schema.CancellationRequest
	err := r.db.WithContext(ctx).
		Where("job_order_id = ? AND status = ?", orderID, schema.CancellationPending).
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *gormRepository) FindCancellationByID(ctx context.Context, id uuid.UUID) (*schema.CancellationRequest, error) {
	var request schema.CancellationRequest
	if err := r.db.WithContext(ctx).First(&request, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *gormRepository) CreateCancellation(ctx context.Context, request *schema.CancellationRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

func (r *gormRepository) UpdateCancellation(ctx context.Context, request *schema.CancellationRequest) error {
	return r.db.WithContext(ctx).Save(request).Error
}
