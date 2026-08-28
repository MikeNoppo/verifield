package joborder_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/modules/joborder"
	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/schema"
)

// fakeRepo adalah Repository di dalam memori. Yang diuji di berkas ini adalah
// aturan bisnisnya, bukan SQL-nya — karena itu Transaction cukup menjalankan
// callback-nya langsung.
type fakeRepo struct {
	order    *schema.JobOrder
	events   []*schema.JobStatusEvent
	requests []*schema.CancellationRequest
	alerts   []*schema.JobOrderAlert
	seq      int64
	notified int64
}

func (f *fakeRepo) Transaction(_ context.Context, fn func(tx joborder.Repository) error) error {
	return fn(f)
}

func (f *fakeRepo) LockOrder(context.Context, uuid.UUID) (*schema.JobOrder, error) {
	return f.order, nil
}

func (f *fakeRepo) InsertEvent(_ context.Context, event *schema.JobStatusEvent) error {
	f.seq++
	event.ID = uuid.New()
	event.Seq = f.seq
	f.events = append(f.events, event)
	return nil
}

func (f *fakeRepo) UpdateOrderStatus(_ context.Context, order *schema.JobOrder, expectedVersion int) error {
	if order.Version != expectedVersion {
		return apperror.Conflict("versi basi")
	}
	order.Version++
	return nil
}

func (f *fakeRepo) InsertAlert(_ context.Context, alert *schema.JobOrderAlert) error {
	f.alerts = append(f.alerts, alert)
	return nil
}

func (f *fakeRepo) Notify(_ context.Context, seq int64, _ uuid.UUID) error {
	f.notified = seq
	return nil
}

func (f *fakeRepo) FindPendingCancellation(_ context.Context, orderID uuid.UUID) (*schema.CancellationRequest, error) {
	for _, r := range f.requests {
		if r.JobOrderID == orderID && r.Status == schema.CancellationPending {
			return r, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) FindCancellationByID(_ context.Context, id uuid.UUID) (*schema.CancellationRequest, error) {
	for _, r := range f.requests {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) CreateCancellation(_ context.Context, request *schema.CancellationRequest) error {
	request.ID = uuid.New()
	f.requests = append(f.requests, request)
	return nil
}

func (f *fakeRepo) UpdateCancellation(context.Context, *schema.CancellationRequest) error {
	return nil
}

func (f *fakeRepo) FindEventByClientID(context.Context, uuid.UUID, string) (*schema.JobStatusEvent, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) FindByIDCompact(context.Context, uuid.UUID) (*schema.JobOrder, dto.Derived, error) {
	return f.order, dto.Derived{}, nil
}

// Sisa kontrak Repository tidak disentuh jalur yang diuji di sini.
func (f *fakeRepo) FindAll(context.Context, dto.ListQuery) ([]schema.JobOrder, map[uuid.UUID]dto.Derived, int64, error) {
	return nil, nil, 0, nil
}

func (f *fakeRepo) FindByID(context.Context, uuid.UUID) (*schema.JobOrder, dto.Derived, error) {
	return f.order, dto.Derived{}, nil
}

func (f *fakeRepo) FindEvents(context.Context, uuid.UUID, int64) ([]schema.JobStatusEvent, error) {
	return nil, nil
}

func (f *fakeRepo) OrderIDsChangedSince(context.Context, int64) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeRepo) UpdateInspector(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (f *fakeRepo) CreateOrder(context.Context, *schema.JobOrder) error { return nil }

func (f *fakeRepo) NextReference(context.Context, int) (string, error) { return "JO-2026-0001", nil }

// ---------------------------------------------------------------------------

type panggung struct {
	repo        *fakeRepo
	svc         joborder.Service
	order       *schema.JobOrder
	request     *schema.CancellationRequest
	inspektor   joborder.Actor
	koordinator joborder.Actor
}

// siapkan membangun satu order pada status tertentu, dengan permintaan
// pembatalan klien yang masih menunggu keputusan koordinator.
func siapkan(t *testing.T, status schema.JobStatus) *panggung {
	t.Helper()

	inspekturID := uuid.New()
	order := &schema.JobOrder{
		ID:              uuid.New(),
		CompanyID:       uuid.New(),
		InspectorID:     &inspekturID,
		CurrentStatus:   status,
		Version:         3,
		CreatedAt:       time.Now().UTC().Add(-4 * time.Hour),
		StatusChangedAt: time.Now().UTC().Add(-time.Hour),
	}
	request := &schema.CancellationRequest{
		ID:            uuid.New(),
		JobOrderID:    order.ID,
		RequestedByID: uuid.New(),
		Reason:        "Pengapalan dimajukan",
		Status:        schema.CancellationPending,
	}
	repo := &fakeRepo{order: order, requests: []*schema.CancellationRequest{request}}

	return &panggung{
		repo:        repo,
		svc:         joborder.NewService(repo, nil),
		order:       order,
		request:     request,
		inspektor:   joborder.Actor{ID: inspekturID, Role: schema.RoleInspector},
		koordinator: joborder.Actor{ID: uuid.New(), Role: schema.RoleAdmin},
	}
}

func (p *panggung) eventTerakhir(t *testing.T) *schema.JobStatusEvent {
	t.Helper()
	if len(p.repo.events) == 0 {
		t.Fatal("tidak ada event yang tercatat")
	}
	return p.repo.events[len(p.repo.events)-1]
}

func statusHTTP(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("error bukan AppError: %v", err)
	}
	return appErr.Status
}

// Keputusan B-10. Inspektor yang sedang offline tidak tahu klien mengajukan
// pembatalan, dan tetap menyelesaikan pekerjaannya. Permintaan itu tidak boleh
// tetap terbuka: menyetujuinya kemudian akan memindahkan order keluar dari
// status final.
func TestPermintaanPembatalanGugurSaatPekerjaanSelesai(t *testing.T) {
	p := siapkan(t, schema.StatusInProgress)

	_, err := p.svc.SubmitEvent(context.Background(), p.inspektor, p.order.ID.String(),
		dto.SubmitStatusEventDTO{ToStatus: "completed", ClientEventID: "penanda-perangkat-1"})
	if err != nil {
		t.Fatalf("laporan selesai seharusnya diterima: %v", err)
	}

	if p.order.CurrentStatus != schema.StatusCompleted {
		t.Errorf("status = %s, ingin completed", p.order.CurrentStatus)
	}
	if p.request.Status != schema.CancellationObsolete {
		t.Errorf("permintaan pembatalan = %s, ingin obsolete", p.request.Status)
	}

	penutup := p.eventTerakhir(t)
	if penutup.Accepted {
		t.Error("entri penutup tidak boleh ikut mengubah status")
	}
	if penutup.RejectionReason == nil || *penutup.RejectionReason != joborder.RejectionCancellationObsolete {
		t.Errorf("alasan penolakan = %v, ingin cancellation_obsolete", penutup.RejectionReason)
	}

	// Ada pekerjaan nyata yang sudah dikerjakan, dan klien terlanjur meminta
	// pembatalan. Keduanya nyata, dan penyelesaian komersialnya tidak boleh
	// hilang diam-diam (keputusan B-07).
	if len(p.repo.alerts) != 1 || p.repo.alerts[0].Type != schema.AlertCancellationObsolete {
		t.Fatalf("koordinator seharusnya diberi tanda, dapat %+v", p.repo.alerts)
	}

	// Kursor real-time harus menunjuk entri terakhir, bukan entri transisinya —
	// kalau tidak, layar koordinator tidak pernah menerima penutupan ini.
	if p.repo.notified != penutup.Seq {
		t.Errorf("seq yang disiarkan = %d, ingin %d", p.repo.notified, penutup.Seq)
	}
}

func TestKeputusanPembatalanDitolakSetelahOrderFinal(t *testing.T) {
	p := siapkan(t, schema.StatusCompleted)

	_, err := p.svc.DecideCancellation(context.Background(), p.koordinator,
		p.order.ID.String(), p.request.ID.String(), dto.DecideCancellationDTO{Decision: "approve"})
	if err == nil {
		t.Fatal("menyetujui pembatalan atas order final seharusnya ditolak")
	}
	if got := statusHTTP(t, err); got != http.StatusConflict {
		t.Errorf("status = %d, ingin %d", got, http.StatusConflict)
	}
	if p.order.CurrentStatus != schema.StatusCompleted {
		t.Errorf("status final berubah menjadi %s", p.order.CurrentStatus)
	}
}

// Koordinator boleh membatalkan langsung sekalipun ada permintaan klien yang
// menunggu. Permintaan itu terpenuhi, bukan gugur — klien mendapat persis yang
// ia minta, dan tidak ada yang perlu ditindaklanjuti.
func TestPembatalanLangsungKoordinatorMemenuhiPermintaanKlien(t *testing.T) {
	p := siapkan(t, schema.StatusInProgress)

	_, err := p.svc.Cancel(context.Background(), p.koordinator, p.order.ID.String(),
		dto.CancelJobOrderDTO{Reason: "Disepakati lewat telepon"})
	if err != nil {
		t.Fatalf("koordinator seharusnya boleh membatalkan: %v", err)
	}

	if p.order.CurrentStatus != schema.StatusCancelled {
		t.Errorf("status = %s, ingin cancelled", p.order.CurrentStatus)
	}
	if p.request.Status != schema.CancellationApproved {
		t.Errorf("permintaan pembatalan = %s, ingin approved", p.request.Status)
	}
	if len(p.repo.alerts) != 0 {
		t.Errorf("tidak perlu ada tanda bagi koordinator, dapat %d", len(p.repo.alerts))
	}
}
