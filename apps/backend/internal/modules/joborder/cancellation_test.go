package joborder_test

import (
	"context"
	"net/http"
	"strings"
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

	// changed adalah order yang "berubah sejak kursor", beserta jumlah
	// pemanggilannya. Jumlah itu yang menjaga jalur pemulihan tetap satu kueri.
	changed      []schema.JobOrder
	changedCalls int
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

func (f *fakeRepo) FindCompactChangedSince(
	context.Context, int64,
) ([]schema.JobOrder, map[uuid.UUID]dto.Derived, error) {
	f.changedCalls++
	return f.changed, map[uuid.UUID]dto.Derived{}, nil
}

func (f *fakeRepo) UpdateInspector(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (f *fakeRepo) CreateOrder(context.Context, *schema.JobOrder) error { return nil }

func (f *fakeRepo) NextReference(context.Context, int) (string, error) { return "JO-2026-0001", nil }

// ---------------------------------------------------------------------------

type fixture struct {
	repo        *fakeRepo
	svc         joborder.Service
	order       *schema.JobOrder
	request     *schema.CancellationRequest
	inspector   joborder.Actor
	coordinator joborder.Actor
}

// siapkan membangun satu order pada status tertentu, dengan permintaan
// pembatalan klien yang masih menunggu keputusan koordinator.
func setup(t *testing.T, status schema.JobStatus) *fixture {
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

	return &fixture{
		repo:        repo,
		svc:         joborder.NewService(repo, nil, joborder.DefaultSchedulePolicy(time.UTC)),
		order:       order,
		request:     request,
		inspector:   joborder.Actor{ID: inspekturID, Role: schema.RoleInspector},
		coordinator: joborder.Actor{ID: uuid.New(), Role: schema.RoleAdmin},
	}
}

func (f *fixture) lastEvent(t *testing.T) *schema.JobStatusEvent {
	t.Helper()
	if len(f.repo.events) == 0 {
		t.Fatal("tidak ada event yang tercatat")
	}
	return f.repo.events[len(f.repo.events)-1]
}

func httpStatus(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("error bukan AppError: %v", err)
	}
	return appErr.Status
}

// Keputusan B-10. Inspektor yang sedang offline tidak tahu klien mengajukan
// pembatalan, dan tetap menyelesaikan pekerjaannya. Permintaan itu tidak boleh
// tetap dapat disetujui — itu akan memindahkan order keluar dari status final —
// tetapi juga tidak boleh dinyatakan selesai, karena pertanyaan komersialnya
// belum dijawab siapa pun.
func TestPermintaanPembatalanMenungguPenyelesaianSaatPekerjaanSelesai(t *testing.T) {
	f := setup(t, schema.StatusInProgress)

	_, err := f.svc.SubmitEvent(context.Background(), f.inspector, f.order.ID.String(),
		dto.SubmitStatusEventDTO{ToStatus: "completed", ClientEventID: "penanda-perangkat-1"})
	if err != nil {
		t.Fatalf("laporan selesai seharusnya diterima: %v", err)
	}

	if f.order.CurrentStatus != schema.StatusCompleted {
		t.Errorf("status = %s, ingin completed", f.order.CurrentStatus)
	}
	if f.request.Status != schema.CancellationPendingSettlement {
		t.Errorf("permintaan pembatalan = %s, ingin pending_settlement", f.request.Status)
	}
	// Belum ada yang memutuskan apa pun; yang berubah hanya pertanyaannya.
	if f.request.DecidedAt != nil || f.request.DecidedByID != nil {
		t.Error("permintaan belum diputuskan, jadi tidak boleh punya pemutus")
	}

	entri := f.lastEvent(t)
	if entri.Accepted {
		t.Error("entri ini tidak boleh ikut mengubah status")
	}
	if entri.RejectionReason == nil || *entri.RejectionReason != joborder.RejectionSettlementPending {
		t.Errorf("alasan = %v, ingin settlement_pending", entri.RejectionReason)
	}

	// Permintaan yang menunggu penyelesaian sudah menjadi tugas di antrean
	// koordinator. Menambah alert berarti dua mekanisme untuk satu keadaan.
	if len(f.repo.alerts) != 0 {
		t.Errorf("tidak perlu alert terpisah, dapat %d", len(f.repo.alerts))
	}

	// Kursor real-time harus menunjuk entri terakhir, bukan entri transisinya —
	// kalau tidak, layar koordinator tidak pernah menerima perubahan ini.
	if f.repo.notified != entri.Seq {
		t.Errorf("seq yang disiarkan = %d, ingin %d", f.repo.notified, entri.Seq)
	}
}

// Inti dari keputusan B-10: koordinator tetap punya tindakan. Yang berubah
// bukan haknya, melainkan pertanyaan yang dihadapkan kepadanya.
func TestPenyelesaianKomersialDicatatKoordinator(t *testing.T) {
	f := setup(t, schema.StatusInProgress)

	if _, err := f.svc.SubmitEvent(context.Background(), f.inspector, f.order.ID.String(),
		dto.SubmitStatusEventDTO{ToStatus: "completed", ClientEventID: "penanda-perangkat-1"}); err != nil {
		t.Fatalf("laporan selesai seharusnya diterima: %v", err)
	}

	_, err := f.svc.SettleCancellation(context.Background(), f.coordinator,
		f.order.ID.String(), f.request.ID.String(),
		dto.SettleCancellationDTO{Outcome: "billed_partial", Note: "Disepakati menagih separuh biaya kunjungan"})
	if err != nil {
		t.Fatalf("koordinator seharusnya dapat memutuskan penyelesaian: %v", err)
	}

	if f.request.Status != schema.CancellationSettled {
		t.Errorf("permintaan = %s, ingin settled", f.request.Status)
	}
	if f.request.SettlementOutcome == nil || *f.request.SettlementOutcome != schema.SettlementBilledPartial {
		t.Errorf("hasil = %v, ingin billed_partial", f.request.SettlementOutcome)
	}
	if f.request.DecidedByID == nil || *f.request.DecidedByID != f.coordinator.ID {
		t.Error("pemutus penyelesaian tidak tercatat")
	}

	// Pekerjaannya memang dikerjakan; keputusan komersial tidak mengubah itu.
	if f.order.CurrentStatus != schema.StatusCompleted {
		t.Errorf("status berubah menjadi %s", f.order.CurrentStatus)
	}

	entri := f.lastEvent(t)
	if entri.RejectionReason == nil || *entri.RejectionReason != joborder.RejectionSettlementDecided {
		t.Errorf("alasan = %v, ingin settlement_decided", entri.RejectionReason)
	}
	if entri.Reason == nil || !strings.Contains(*entri.Reason, "Ditagih sebagian") {
		t.Errorf("riwayat harus memuat hasilnya, dapat %v", entri.Reason)
	}
}

// Selama pekerjaan masih berjalan, yang berlaku adalah keputusan pembatalan —
// bukan penyelesaian. Dua jalur ini tidak boleh saling menggantikan.
func TestPenyelesaianDitolakSelamaPekerjaanMasihBerjalan(t *testing.T) {
	f := setup(t, schema.StatusInProgress)

	_, err := f.svc.SettleCancellation(context.Background(), f.coordinator,
		f.order.ID.String(), f.request.ID.String(),
		dto.SettleCancellationDTO{Outcome: "waived", Note: "Coba menyelesaikan terlalu dini"})
	if err == nil {
		t.Fatal("penyelesaian atas permintaan yang masih menunggu keputusan seharusnya ditolak")
	}
	if got := httpStatus(t, err); got != http.StatusConflict {
		t.Errorf("status = %d, ingin %d", got, http.StatusConflict)
	}
	if f.request.Status != schema.CancellationPending {
		t.Errorf("permintaan berubah menjadi %s", f.request.Status)
	}
}

func TestKeputusanPembatalanDitolakSetelahOrderFinal(t *testing.T) {
	f := setup(t, schema.StatusCompleted)

	_, err := f.svc.DecideCancellation(context.Background(), f.coordinator,
		f.order.ID.String(), f.request.ID.String(), dto.DecideCancellationDTO{Decision: "approve"})
	if err == nil {
		t.Fatal("menyetujui pembatalan atas order final seharusnya ditolak")
	}
	if got := httpStatus(t, err); got != http.StatusConflict {
		t.Errorf("status = %d, ingin %d", got, http.StatusConflict)
	}
	if f.order.CurrentStatus != schema.StatusCompleted {
		t.Errorf("status final berubah menjadi %s", f.order.CurrentStatus)
	}
}

// Koordinator boleh membatalkan langsung sekalipun ada permintaan klien yang
// menunggu. Permintaan itu terpenuhi, bukan gugur — klien mendapat persis yang
// ia minta, dan tidak ada yang perlu ditindaklanjuti.
func TestPembatalanLangsungKoordinatorMemenuhiPermintaanKlien(t *testing.T) {
	f := setup(t, schema.StatusInProgress)

	_, err := f.svc.Cancel(context.Background(), f.coordinator, f.order.ID.String(),
		dto.CancelJobOrderDTO{Reason: "Disepakati lewat telepon"})
	if err != nil {
		t.Fatalf("koordinator seharusnya boleh membatalkan: %v", err)
	}

	if f.order.CurrentStatus != schema.StatusCancelled {
		t.Errorf("status = %s, ingin cancelled", f.order.CurrentStatus)
	}
	if f.request.Status != schema.CancellationApproved {
		t.Errorf("permintaan pembatalan = %s, ingin approved", f.request.Status)
	}
	if len(f.repo.alerts) != 0 {
		t.Errorf("tidak perlu ada tanda bagi koordinator, dapat %d", len(f.repo.alerts))
	}
}

// Penolakan pembatalan punya dua sebab yang berbeda, dan keduanya harus dapat
// dibedakan tanpa membaca kalimatnya: wewenang yang tidak pernah ada menjawab
// 403, sedangkan keadaan yang kebetulan tidak mengizinkan menjawab 409. Klien
// yang menerima 403 menyembunyikan tombolnya; yang menerima 409 memuat ulang.
func TestPenolakanPembatalanMembedakanWewenangDariKeadaan(t *testing.T) {
	f := setup(t, schema.StatusOnSite)

	_, err := f.svc.Cancel(context.Background(), f.inspector, f.order.ID.String(),
		dto.CancelJobOrderDTO{Reason: "Lokasinya terlalu jauh"})
	if got := httpStatus(t, err); got != http.StatusForbidden {
		t.Errorf("inspektor membatalkan: status = %d, ingin %d", got, http.StatusForbidden)
	}

	f.order.CurrentStatus = schema.StatusCompleted

	_, err = f.svc.Cancel(context.Background(), f.coordinator, f.order.ID.String(),
		dto.CancelJobOrderDTO{Reason: "Diminta klien lewat telepon"})
	if got := httpStatus(t, err); got != http.StatusConflict {
		t.Errorf("koordinator membatalkan order final: status = %d, ingin %d", got, http.StatusConflict)
	}
}

// Pemulihan berjalan justru saat banyak klien menyambung ulang bersamaan —
// sesudah rilis, atau sesudah jaringan pulih. Satu kueri per order pada jalur
// itu berarti beban terbesar tepat pada saat sistem paling rapuh.
//
// Urutannya juga dijaga: field id pada pesan SSE diisi seq, dan browser
// mengirim balik nilai TERAKHIR yang ia terima sebagai Last-Event-ID.
func TestPemulihanMemakaiSatuKueriDanMenjagaUrutan(t *testing.T) {
	f := setup(t, schema.StatusInProgress)
	f.repo.changed = []schema.JobOrder{
		{ID: uuid.New(), ReferenceNumber: "JO-2026-0007"},
		{ID: uuid.New(), ReferenceNumber: "JO-2026-0002"},
		{ID: uuid.New(), ReferenceNumber: "JO-2026-0005"},
	}

	snapshots, err := f.svc.SnapshotsChangedSince(context.Background(), 12)
	if err != nil {
		t.Fatalf("pemulihan gagal: %v", err)
	}

	if f.repo.changedCalls != 1 {
		t.Errorf("jumlah kueri pemulihan = %d, mau 1", f.repo.changedCalls)
	}

	if len(snapshots) != len(f.repo.changed) {
		t.Fatalf("jumlah snapshot = %d, mau %d", len(snapshots), len(f.repo.changed))
	}
	for i, want := range f.repo.changed {
		if snapshots[i].ReferenceNumber != want.ReferenceNumber {
			t.Errorf("snapshot ke-%d = %s, mau %s",
				i, snapshots[i].ReferenceNumber, want.ReferenceNumber)
		}
	}
}
