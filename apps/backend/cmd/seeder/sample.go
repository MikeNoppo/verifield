package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"verifield-be/internal/schema"
	"verifield-be/internal/shared/hash"
)

const refScopeJobOrder = "job_order"

// seedSample mengisi data contoh supaya sistem bisa langsung dicoba (dokumen
// konteks bisnis bagian 13). Idempoten lewat pengecekan kunci alami, jadi aman
// dijalankan berulang kali.
func seedSample(ctx context.Context, db *gorm.DB, password string, log *slog.Logger) error {
	types, err := seedInspectionTypes(ctx, db, log)
	if err != nil {
		return fmt.Errorf("jenis inspeksi: %w", err)
	}

	companies, err := seedCompanies(ctx, db, log)
	if err != nil {
		return fmt.Errorf("perusahaan: %w", err)
	}

	people, err := seedPeople(ctx, db, companies, password, log)
	if err != nil {
		return fmt.Errorf("pengguna: %w", err)
	}

	if err := seedJobOrders(ctx, db, types, companies, people, log); err != nil {
		return fmt.Errorf("job order: %w", err)
	}
	return nil
}

func seedInspectionTypes(ctx context.Context, db *gorm.DB, log *slog.Logger) (map[string]*schema.InspectionType, error) {
	wanted := []schema.InspectionType{
		{Code: "bulk_cargo", Name: "Inspeksi Kargo Curah"},
		{Code: "container", Name: "Inspeksi Kontainer"},
		{Code: "tank", Name: "Kalibrasi & Inspeksi Tangki"},
		{Code: "stockpile", Name: "Survei Stockpile"},
		{Code: "agriculture", Name: "Inspeksi Hasil Pertanian"},
	}

	result := make(map[string]*schema.InspectionType, len(wanted))
	for _, w := range wanted {
		row := w
		row.IsActive = true
		if err := upsertByColumn(ctx, db, &row, "code", row.Code); err != nil {
			return nil, err
		}
		result[row.Code] = &row
	}

	log.Info("jenis inspeksi siap", "jumlah", len(result))
	return result, nil
}

func seedCompanies(ctx context.Context, db *gorm.DB, log *slog.Logger) (map[string]*schema.Company, error) {
	wanted := []schema.Company{
		{Code: "SEN", Name: "PT Samudra Ekspor Nusantara"},
		{Code: "BTP", Name: "PT Bara Timur Perkasa"},
	}

	result := make(map[string]*schema.Company, len(wanted))
	for _, w := range wanted {
		row := w
		row.IsActive = true
		if err := upsertByColumn(ctx, db, &row, "code", row.Code); err != nil {
			return nil, err
		}
		result[row.Code] = &row
	}

	log.Info("perusahaan klien siap", "jumlah", len(result))
	return result, nil
}

type sampleUsers struct {
	clientSEN  *schema.User
	clientBTP  *schema.User
	inspector  *schema.User
	inspector2 *schema.User
	cs         *schema.User
}

// seedPeople memakai password yang sama dengan SEED_ADMIN_PASSWORD supaya tidak
// ada kredensial contoh yang tertulis di dalam repositori.
func seedPeople(ctx context.Context, db *gorm.DB, companies map[string]*schema.Company, password string, log *slog.Logger) (*sampleUsers, error) {
	hashed, err := hash.Password(password)
	if err != nil {
		return nil, err
	}

	mk := func(name, email string, role schema.Role, company *schema.Company) (*schema.User, error) {
		row := schema.User{
			Name:     name,
			Email:    email,
			Password: hashed,
			Role:     role,
			IsActive: true,
		}
		if company != nil {
			row.CompanyID = &company.ID
		}
		if err := upsertByColumn(ctx, db, &row, "email", email); err != nil {
			return nil, err
		}
		return &row, nil
	}

	out := &sampleUsers{}
	specs := []struct {
		target  **schema.User
		name    string
		email   string
		role    schema.Role
		company *schema.Company
	}{
		{&out.clientSEN, "Budi Santoso", "budi@sen.co.id", schema.RoleClient, companies["SEN"]},
		{&out.clientBTP, "Dewi Lestari", "dewi@btp.co.id", schema.RoleClient, companies["BTP"]},
		{&out.inspector, "Joko Prasetyo", "joko@verifield.id", schema.RoleInspector, nil},
		{&out.inspector2, "Rina Amelia", "rina@verifield.id", schema.RoleInspector, nil},
		{&out.cs, "Sari Wijaya", "sari@verifield.id", schema.RoleCS, nil},
	}

	for _, s := range specs {
		u, err := mk(s.name, s.email, s.role, s.company)
		if err != nil {
			return nil, err
		}
		*s.target = u
	}

	log.Info("pengguna contoh siap", "jumlah", len(specs))
	return out, nil
}

func seedJobOrders(
	ctx context.Context,
	db *gorm.DB,
	types map[string]*schema.InspectionType,
	companies map[string]*schema.Company,
	people *sampleUsers,
	log *slog.Logger,
) error {
	var existing int64
	if err := db.WithContext(ctx).Model(&schema.JobOrder{}).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		log.Info("job order contoh sudah ada, dilewati", "jumlah", existing)
		return nil
	}

	// Koordinator yang menugaskan diambil dari akun admin hasil tahap pertama.
	var coordinator schema.User
	if err := db.WithContext(ctx).Where("role = ?", schema.RoleAdmin).First(&coordinator).Error; err != nil {
		return fmt.Errorf("akun admin belum ada: %w", err)
	}

	now := time.Now().Truncate(time.Minute)

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Selesai penuh — memperlihatkan riwayat lengkap dari Requested sampai Completed.
		done, err := newJobOrder(tx, jobOrderSpec{
			company:     companies["SEN"],
			createdBy:   people.clientSEN,
			typ:         types["bulk_cargo"],
			object:      "Batu bara 5.000 MT",
			locName:     "Dermaga 3, Pelabuhan Tanjung Priok",
			locAddress:  "Jl. Palmerah No. 1, Tanjung Priok, Jakarta Utara",
			city:        "Jakarta",
			startAt:     now.Add(-30 * time.Hour),
			endAt:       now.Add(-26 * time.Hour),
			requestedAt: now.Add(-48 * time.Hour),
		})
		if err != nil {
			return err
		}
		steps := []struct {
			to      schema.JobStatus
			actor   *schema.User
			at      time.Time
			comment string
		}{
			{schema.StatusAssigned, &coordinator, now.Add(-40 * time.Hour), ""},
			{schema.StatusOnTheWay, people.inspector, now.Add(-32 * time.Hour), ""},
			{schema.StatusOnSite, people.inspector, now.Add(-30 * time.Hour), ""},
			{schema.StatusInProgress, people.inspector, now.Add(-29 * time.Hour), ""},
			{schema.StatusCompleted, people.inspector, now.Add(-26 * time.Hour), ""},
		}
		if err := tx.Model(done).Update("inspector_id", people.inspector.ID).Error; err != nil {
			return err
		}
		for _, s := range steps {
			if err := applyEvent(tx, done, s.to, s.actor, s.at, s.comment); err != nil {
				return err
			}
		}

		// 2. Sedang berjalan — kasus paling sering dilihat klien di layar.
		running, err := newJobOrder(tx, jobOrderSpec{
			company:     companies["BTP"],
			createdBy:   people.clientBTP,
			typ:         types["stockpile"],
			object:      "Stockpile batu bara blok C",
			locName:     "Site Sangatta",
			locAddress:  "Jl. Tambang Raya KM 12, Sangatta, Kutai Timur",
			city:        "Sangatta",
			startAt:     now.Add(-4 * time.Hour),
			endAt:       now.Add(2 * time.Hour),
			requestedAt: now.Add(-20 * time.Hour),
		})
		if err != nil {
			return err
		}
		if err := tx.Model(running).Update("inspector_id", people.inspector2.ID).Error; err != nil {
			return err
		}
		for _, s := range []struct {
			to    schema.JobStatus
			actor *schema.User
			at    time.Time
		}{
			{schema.StatusAssigned, &coordinator, now.Add(-18 * time.Hour)},
			{schema.StatusOnTheWay, people.inspector2, now.Add(-6 * time.Hour)},
			{schema.StatusOnSite, people.inspector2, now.Add(-4 * time.Hour)},
			{schema.StatusInProgress, people.inspector2, now.Add(-3 * time.Hour)},
		} {
			if err := applyEvent(tx, running, s.to, s.actor, s.at, ""); err != nil {
				return err
			}
		}

		// 3. Baru masuk, belum ada inspektor.
		if _, err := newJobOrder(tx, jobOrderSpec{
			company:     companies["SEN"],
			createdBy:   people.clientSEN,
			typ:         types["container"],
			object:      "40 kontainer CPO",
			locName:     "Terminal Peti Kemas Belawan",
			locAddress:  "Jl. Pelabuhan Belawan, Medan",
			city:        "Medan",
			startAt:     now.Add(24 * time.Hour),
			endAt:       now.Add(30 * time.Hour),
			requestedAt: now.Add(-2 * time.Hour),
		}); err != nil {
			return err
		}

		// 4. Dibatalkan klien saat inspektor offline, lalu laporan "selesai" telat
		//    masuk. Memperlihatkan keputusan B-07: status tidak berubah, event tetap
		//    dicatat, dan koordinator diberi tanda.
		cancelled, err := newJobOrder(tx, jobOrderSpec{
			company:     companies["BTP"],
			createdBy:   people.clientBTP,
			typ:         types["tank"],
			object:      "Tangki timbun T-401",
			locName:     "Terminal BBM Balikpapan",
			locAddress:  "Jl. Minyak No. 4, Balikpapan",
			city:        "Balikpapan",
			startAt:     now.Add(-10 * time.Hour),
			endAt:       now.Add(-6 * time.Hour),
			requestedAt: now.Add(-26 * time.Hour),
		})
		if err != nil {
			return err
		}
		if err := tx.Model(cancelled).Update("inspector_id", people.inspector.ID).Error; err != nil {
			return err
		}
		for _, s := range []struct {
			to      schema.JobStatus
			actor   *schema.User
			at      time.Time
			comment string
		}{
			{schema.StatusAssigned, &coordinator, now.Add(-24 * time.Hour), ""},
			{schema.StatusOnTheWay, people.inspector, now.Add(-12 * time.Hour), ""},
			{schema.StatusCancelled, people.clientBTP, now.Add(-11 * time.Hour), "Kargo belum tiba, klien membatalkan"},
		} {
			if err := applyEvent(tx, cancelled, s.to, s.actor, s.at, s.comment); err != nil {
				return err
			}
		}
		if err := recordLateRejected(tx, cancelled, schema.StatusCompleted, people.inspector, now.Add(-7*time.Hour)); err != nil {
			return err
		}

		log.Info("job order contoh dibuat", "jumlah", 4)
		return nil
	})
}

type jobOrderSpec struct {
	company     *schema.Company
	createdBy   *schema.User
	typ         *schema.InspectionType
	object      string
	locName     string
	locAddress  string
	city        string
	startAt     time.Time
	endAt       time.Time
	requestedAt time.Time
}

// newJobOrder membuat order pada status Requested beserta event pertamanya.
func newJobOrder(tx *gorm.DB, s jobOrderSpec) (*schema.JobOrder, error) {
	ref, err := nextReference(tx, s.requestedAt.Year())
	if err != nil {
		return nil, err
	}

	order := &schema.JobOrder{
		ReferenceNumber:   ref,
		CompanyID:         s.company.ID,
		CreatedByID:       s.createdBy.ID,
		InspectionTypeID:  s.typ.ID,
		ObjectDescription: s.object,
		LocationName:      s.locName,
		LocationAddress:   s.locAddress,
		City:              s.city,
		ScheduledStartAt:  s.startAt,
		ScheduledEndAt:    s.endAt,
		CurrentStatus:     schema.StatusRequested,
		StatusChangedAt:   s.requestedAt,
	}
	if err := tx.Create(order).Error; err != nil {
		return nil, err
	}

	// FromStatus kosong karena ini event pertama. Aktornya diisi klien pemesan,
	// bukan dikosongkan, supaya jelas siapa yang memicu order ini.
	first := schema.JobStatusEvent{
		JobOrderID: order.ID,
		ToStatus:   schema.StatusRequested,
		ActorRole:  schema.RoleClient,
		ActorID:    &s.createdBy.ID,
		OccurredAt: s.requestedAt,
		ReceivedAt: s.requestedAt,
		Accepted:   true,
	}
	if err := tx.Create(&first).Error; err != nil {
		return nil, err
	}
	return order, nil
}

// applyEvent menyisipkan event yang diterima DAN memperbarui cache status di
// job_orders dalam transaksi yang sama — invarian keputusan B-01.
func applyEvent(tx *gorm.DB, order *schema.JobOrder, to schema.JobStatus, actor *schema.User, at time.Time, reason string) error {
	from := order.CurrentStatus
	event := schema.JobStatusEvent{
		JobOrderID: order.ID,
		FromStatus: &from,
		ToStatus:   to,
		ActorID:    &actor.ID,
		ActorRole:  actor.Role,
		OccurredAt: at,
		ReceivedAt: at,
		Accepted:   true,
	}
	if reason != "" {
		event.Reason = &reason
	}
	if err := tx.Create(&event).Error; err != nil {
		return err
	}

	order.CurrentStatus = to
	order.StatusChangedAt = at
	order.Version++

	return tx.Model(order).Updates(map[string]any{
		"current_status":    to,
		"status_changed_at": at,
		"version":           order.Version,
	}).Error
}

// recordLateRejected mencatat pembaruan yang datang setelah status final:
// statusnya tidak berubah, tetapi event tetap tersimpan dan koordinator diberi
// tanda (keputusan B-07).
func recordLateRejected(tx *gorm.DB, order *schema.JobOrder, to schema.JobStatus, actor *schema.User, at time.Time) error {
	from := order.CurrentStatus
	rejection := "late_after_final"
	event := schema.JobStatusEvent{
		JobOrderID:      order.ID,
		FromStatus:      &from,
		ToStatus:        to,
		ActorID:         &actor.ID,
		ActorRole:       actor.Role,
		OccurredAt:      at,
		ReceivedAt:      at.Add(90 * time.Minute),
		Accepted:        false,
		RejectionReason: &rejection,
	}
	if err := tx.Create(&event).Error; err != nil {
		return err
	}

	alert := schema.JobOrderAlert{
		JobOrderID:    order.ID,
		Type:          schema.AlertLateUpdateRejected,
		SourceEventID: &event.ID,
		Message: fmt.Sprintf(
			"Inspektor melaporkan %q setelah order berstatus %s. Pekerjaan lapangan sudah terlanjur dikerjakan dan perlu diselesaikan kompensasinya.",
			to, from,
		),
	}
	return tx.Create(&alert).Error
}

// nextReference mengalokasikan nomor urut berikutnya dalam satu pernyataan
// atomik, bukan COUNT(*), supaya dua transaksi tidak pernah mendapat nomor yang
// sama.
func nextReference(tx *gorm.DB, year int) (string, error) {
	var number int64
	err := tx.Raw(`
		INSERT INTO reference_counters (scope, year, last_number, updated_at)
		VALUES (?, ?, 1, now())
		ON CONFLICT (scope, year) DO UPDATE
			SET last_number = reference_counters.last_number + 1, updated_at = now()
		RETURNING last_number`,
		refScopeJobOrder, year,
	).Scan(&number).Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("JO-%d-%04d", year, number), nil
}

// upsertByColumn membuat baris kalau kunci alaminya belum ada, dan mengisi
// struct dengan baris yang sudah ada kalau sudah.
func upsertByColumn(ctx context.Context, db *gorm.DB, dest any, column string, value any) error {
	err := db.WithContext(ctx).Where(column+" = ?", value).First(dest).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.WithContext(ctx).Create(dest).Error
}
