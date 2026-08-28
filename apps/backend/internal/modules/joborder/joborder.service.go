package joborder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/pagination"
	"verifield-be/internal/common/response"
	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/schema"
)

// Actor adalah pihak yang melakukan aksi. Diserahkan eksplisit sebagai
// parameter, bukan digali dari context di dalam service, supaya business logic
// bisa diuji tanpa gin dan bisa dipakai ulang oleh seeder.
type Actor struct {
	ID        uuid.UUID
	Name      string
	Role      schema.Role
	CompanyID *uuid.UUID
}

// UserFinder adalah bagian kecil dari modul user yang dibutuhkan modul ini.
// Dideklarasikan di sisi pemakai supaya ketergantungannya sesempit mungkin.
type UserFinder interface {
	FindEntityByID(ctx context.Context, id string) (*schema.User, error)
}

// Service berisi business logic job order.
type Service interface {
	List(ctx context.Context, actor Actor, query dto.ListQuery) ([]dto.JobOrderResponse, response.Meta, error)
	Detail(ctx context.Context, actor Actor, id string) (*dto.JobOrderResponse, error)
	Events(ctx context.Context, actor Actor, id string, afterSeq int64) ([]dto.JobStatusEventResponse, error)

	Create(ctx context.Context, actor Actor, input dto.CreateJobOrderDTO) (*dto.JobOrderResponse, error)
	Assign(ctx context.Context, actor Actor, id string, input dto.AssignInspectorDTO) (*dto.JobOrderResponse, error)
	SubmitEvent(ctx context.Context, actor Actor, id string, input dto.SubmitStatusEventDTO) (*dto.SubmitStatusEventResult, error)
	Cancel(ctx context.Context, actor Actor, id string, input dto.CancelJobOrderDTO) (*dto.CancelResult, error)
	DecideCancellation(ctx context.Context, actor Actor, id, requestID string, input dto.DecideCancellationDTO) (*dto.JobOrderResponse, error)
	SettleCancellation(ctx context.Context, actor Actor, id, requestID string, input dto.SettleCancellationDTO) (*dto.JobOrderResponse, error)
	Correct(ctx context.Context, actor Actor, id string, input dto.CorrectStatusDTO) (*dto.JobOrderResponse, error)

	// Dipakai lapisan real-time untuk menyusun payload yang dikirim ke klien.
	Snapshot(ctx context.Context, orderID uuid.UUID) (*dto.JobOrderResponse, error)
	SnapshotsChangedSince(ctx context.Context, seq int64) ([]dto.JobOrderResponse, error)
}

type service struct {
	repo  Repository
	users UserFinder
}

// NewService merakit service dari repository dan dependensi lintas modulnya.
func NewService(repo Repository, users UserFinder) Service {
	return &service{repo: repo, users: users}
}

// ---------------------------------------------------------------------------
// Baca
// ---------------------------------------------------------------------------

func (s *service) List(ctx context.Context, actor Actor, query dto.ListQuery) ([]dto.JobOrderResponse, response.Meta, error) {
	query.Normalize(SortableColumns, "status_changed_at")

	query, err := ScopeQuery(actor, query)
	if err != nil {
		return nil, response.Meta{}, err
	}

	orders, derived, total, err := s.repo.FindAll(ctx, query)
	if err != nil {
		return nil, response.Meta{}, err
	}

	out := make([]dto.JobOrderResponse, 0, len(orders))
	for i := range orders {
		out = append(out, dto.ToJobOrderResponse(&orders[i], derived[orders[i].ID]))
	}

	return out, pagination.BuildMeta(query.Query, total), nil
}

func (s *service) Detail(ctx context.Context, actor Actor, id string) (*dto.JobOrderResponse, error) {
	order, derived, err := s.findVisible(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	res := dto.ToJobOrderResponse(order, derived)
	return &res, nil
}

func (s *service) Events(ctx context.Context, actor Actor, id string, afterSeq int64) ([]dto.JobStatusEventResponse, error) {
	order, _, err := s.findVisible(ctx, actor, id)
	if err != nil {
		return nil, err
	}

	events, err := s.repo.FindEvents(ctx, order.ID, afterSeq)
	if err != nil {
		return nil, err
	}
	return dto.ToEventResponses(events), nil
}

// findVisible memuat order sekaligus menegakkan batas kepemilikan.
//
// Order di luar batas dijawab "tidak ditemukan", bukan "tidak boleh":
// membedakan keduanya membocorkan keberadaan order milik klien lain, yang
// justru merupakan informasi komersial (asumsi A-03).
func (s *service) findVisible(ctx context.Context, actor Actor, id string) (*schema.JobOrder, dto.Derived, error) {
	orderID, err := parseID(id)
	if err != nil {
		return nil, dto.Derived{}, err
	}

	order, derived, err := s.repo.FindByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, dto.Derived{}, apperror.NotFound("Job order tidak ditemukan")
		}
		return nil, dto.Derived{}, err
	}

	if !VisibleTo(actor, order) {
		return nil, dto.Derived{}, apperror.NotFound("Job order tidak ditemukan")
	}

	return order, derived, nil
}

// Snapshot menyusun keadaan terkini satu order TANPA riwayatnya.
//
// Riwayat sengaja tidak ikut: pesan real-time dikirim ke setiap klien pada
// setiap perubahan, dan menyertakan seluruh riwayat di sana berarti mengirim
// ulang data yang sudah dimiliki klien berkali-kali. Layar detail melengkapi
// timeline-nya lewat GET /orders/{id}/events?after_seq=, yang hanya mengirim
// selisihnya.
func (s *service) Snapshot(ctx context.Context, orderID uuid.UUID) (*dto.JobOrderResponse, error) {
	order, derived, err := s.repo.FindByIDCompact(ctx, orderID)
	if err != nil {
		return nil, err
	}
	res := dto.ToJobOrderResponse(order, derived)
	return &res, nil
}

func (s *service) SnapshotsChangedSince(ctx context.Context, seq int64) ([]dto.JobOrderResponse, error) {
	ids, err := s.repo.OrderIDsChangedSince(ctx, seq)
	if err != nil {
		return nil, err
	}

	out := make([]dto.JobOrderResponse, 0, len(ids))
	for _, id := range ids {
		snapshot, err := s.Snapshot(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, *snapshot)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Tulis
// ---------------------------------------------------------------------------

func (s *service) Create(ctx context.Context, actor Actor, input dto.CreateJobOrderDTO) (*dto.JobOrderResponse, error) {
	if actor.Role != schema.RoleClient {
		return nil, apperror.Forbidden("Hanya klien yang dapat membuat permintaan inspeksi")
	}
	if actor.CompanyID == nil {
		return nil, apperror.Forbidden("Akun klien ini belum terhubung ke perusahaan mana pun")
	}

	typeID, err := parseUUIDField(input.InspectionTypeID, "inspection_type_id")
	if err != nil {
		return nil, err
	}

	var created *schema.JobOrder
	err = s.repo.Transaction(ctx, func(tx Repository) error {
		now := time.Now().UTC()

		reference, err := tx.NextReference(ctx, now.Year())
		if err != nil {
			return err
		}

		order := &schema.JobOrder{
			ReferenceNumber:   reference,
			CompanyID:         *actor.CompanyID,
			CreatedByID:       actor.ID,
			InspectionTypeID:  typeID,
			ObjectDescription: input.ObjectDescription,
			LocationName:      input.LocationName,
			LocationAddress:   input.LocationAddress,
			City:              input.City,
			ScheduledStartAt:  input.ScheduledStartAt,
			ScheduledEndAt:    input.ScheduledEndAt,
			CurrentStatus:     schema.StatusRequested,
			StatusChangedAt:   now,
		}
		if err := tx.CreateOrder(ctx, order); err != nil {
			return err
		}

		// FromStatus kosong karena ini event pertama. Aktornya diisi klien
		// pemesan, bukan dikosongkan, supaya jelas siapa yang memicu order ini.
		event := &schema.JobStatusEvent{
			JobOrderID: order.ID,
			ToStatus:   schema.StatusRequested,
			ActorID:    &actor.ID,
			ActorRole:  actor.Role,
			OccurredAt: now,
			ReceivedAt: now,
			Accepted:   true,
		}
		if err := tx.InsertEvent(ctx, event); err != nil {
			return err
		}

		created = order
		return tx.Notify(ctx, event.Seq, order.ID)
	})
	if err != nil {
		return nil, err
	}

	return s.Snapshot(ctx, created.ID)
}

func (s *service) Assign(ctx context.Context, actor Actor, id string, input dto.AssignInspectorDTO) (*dto.JobOrderResponse, error) {
	if actor.Role != schema.RoleAdmin {
		return nil, apperror.Forbidden("Hanya koordinator yang dapat menugaskan inspektor")
	}

	orderID, err := parseID(id)
	if err != nil {
		return nil, err
	}

	inspector, err := s.users.FindEntityByID(ctx, input.InspectorID)
	if err != nil {
		return nil, err
	}
	if inspector.Role != schema.RoleInspector {
		return nil, apperror.BadRequest("Pengguna yang dipilih bukan inspektor")
	}
	if !inspector.IsActive {
		return nil, apperror.BadRequest("Inspektor ini sedang tidak aktif")
	}

	err = s.repo.Transaction(ctx, func(tx Repository) error {
		order, err := lockOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if err := ensureVersion(order, input.ExpectedVersion); err != nil {
			return err
		}
		if order.CurrentStatus != schema.StatusRequested {
			return apperror.Conflict(fmt.Sprintf(
				"Order ini sudah %s, sehingga tidak bisa ditugaskan lagi dari layar ini.",
				StatusLabel(order.CurrentStatus),
			))
		}

		if err := tx.UpdateInspector(ctx, order.ID, inspector.ID); err != nil {
			return err
		}

		now := time.Now().UTC()
		event, err := insertTransition(ctx, tx, order, schema.StatusAssigned, actor, now, now, nil)
		if err != nil {
			return err
		}

		order.CurrentStatus = schema.StatusAssigned
		order.StatusChangedAt = now
		if err := tx.UpdateOrderStatus(ctx, order, input.ExpectedVersion); err != nil {
			return err
		}

		return tx.Notify(ctx, event.Seq, order.ID)
	})
	if err != nil {
		return nil, err
	}

	return s.Snapshot(ctx, orderID)
}

func (s *service) SubmitEvent(ctx context.Context, actor Actor, id string, input dto.SubmitStatusEventDTO) (*dto.SubmitStatusEventResult, error) {
	if actor.Role != schema.RoleInspector {
		return nil, apperror.Forbidden("Hanya inspektor yang dapat memperbarui status dari lapangan")
	}

	orderID, err := parseID(id)
	if err != nil {
		return nil, err
	}

	var (
		event     *schema.JobStatusEvent
		duplicate bool
	)

	err = s.repo.Transaction(ctx, func(tx Repository) error {
		order, err := lockOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if !VisibleTo(actor, order) {
			return apperror.Forbidden("Order ini tidak ditugaskan kepada Anda")
		}

		// Keputusan B-03. Pemeriksaan ini bebas balapan karena baris order sudah
		// dikunci FOR UPDATE, sehingga seluruh penulisan event untuk satu order
		// berjalan berurutan. Unique index tetap ada sebagai jaring pengaman.
		existing, err := tx.FindEventByClientID(ctx, orderID, input.ClientEventID)
		switch {
		case err == nil:
			event, duplicate = existing, true
			return nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		receivedAt := time.Now().UTC()
		var reported time.Time
		if input.OccurredAt != nil {
			reported = input.OccurredAt.UTC()
		}
		occurredAt, adjusted := ClampOccurredAt(reported, receivedAt, order.CreatedAt)

		from := order.CurrentStatus
		to := schema.JobStatus(input.ToStatus)

		event = &schema.JobStatusEvent{
			JobOrderID:         order.ID,
			FromStatus:         &from,
			ToStatus:           to,
			ActorID:            &actor.ID,
			ActorRole:          actor.Role,
			OccurredAt:         occurredAt,
			ReceivedAt:         receivedAt,
			OccurredAtAdjusted: adjusted,
			ClientEventID:      &input.ClientEventID,
			Accepted:           true,
			Reason:             input.Reason,
		}

		switch {
		case from.IsFinal():
			// Keputusan B-07: statusnya tidak berubah, tetapi laporannya tidak
			// boleh hilang — ada pekerjaan nyata yang sudah dilakukan seseorang.
			event.Accepted = false
			event.RejectionReason = ptr(RejectionLateAfterFinal)
		case !CanTransition(from, to):
			// Keputusan B-06: pembaruan yang datang keluar urutan tidak boleh
			// membuat status yang dilihat klien mundur.
			event.Accepted = false
			event.RejectionReason = ptr(RejectionOutOfOrder)
		}

		if err := tx.InsertEvent(ctx, event); err != nil {
			return err
		}

		notifySeq := event.Seq

		if event.Accepted {
			order.CurrentStatus = to
			// Waktu kejadian di lapangan, bukan waktu terima. Laporan yang
			// tertahan tiga jam memang membuat order langsung tampak "tanpa
			// pembaruan" di layar koordinator — dan itu benar, karena kabar
			// terakhir dari lapangan memang berumur tiga jam.
			order.StatusChangedAt = occurredAt
			if err := tx.UpdateOrderStatus(ctx, order, order.Version); err != nil {
				return err
			}

			// Keputusan B-10. Inspektor yang sedang offline tidak tahu klien
			// mengajukan pembatalan, dan tetap menyelesaikan pekerjaannya.
			if to.IsFinal() {
				closing, err := holdCancellationForSettlement(ctx, tx, order, actor, receivedAt)
				if err != nil {
					return err
				}
				if closing != nil {
					notifySeq = closing.Seq
				}
			}
		} else if from.IsFinal() {
			alert := &schema.JobOrderAlert{
				JobOrderID:    order.ID,
				Type:          schema.AlertLateUpdateRejected,
				SourceEventID: &event.ID,
				Message: fmt.Sprintf(
					"Inspektor melaporkan %q setelah order berstatus %s. Pekerjaan lapangan sudah terlanjur dikerjakan dan perlu diselesaikan kompensasinya.",
					StatusLabel(to), StatusLabel(from),
				),
			}
			if err := tx.InsertAlert(ctx, alert); err != nil {
				return err
			}
		}

		return tx.Notify(ctx, notifySeq, order.ID)
	})
	if err != nil {
		return nil, err
	}

	snapshot, err := s.Snapshot(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return &dto.SubmitStatusEventResult{
		Accepted:  event.Accepted,
		Duplicate: duplicate,
		Message:   submitMessage(event, duplicate),
		Event:     dto.ToEventResponse(event),
		Order:     *snapshot,
	}, nil
}

// submitMessage menyusun kalimat yang dibaca inspektor di lapangan. Penolakan
// harus terdengar seperti penjelasan, bukan kegagalan sistem — pengguna yang
// laporannya "ditelan" tanpa penjelasan akan kembali melapor lewat telepon.
func submitMessage(event *schema.JobStatusEvent, duplicate bool) string {
	if duplicate {
		return "Laporan ini sudah kami terima sebelumnya. Tidak ada yang tercatat dua kali."
	}
	if event.Accepted {
		return "Status berhasil diperbarui."
	}

	if event.RejectionReason != nil && *event.RejectionReason == RejectionLateAfterFinal {
		return "Order sudah ditutup sebelum laporan Anda masuk, jadi statusnya tidak berubah. Laporan Anda tetap tercatat dan koordinator sudah diberi tahu untuk menindaklanjuti."
	}
	return "Status order sudah lebih maju daripada laporan ini, jadi statusnya tidak diubah. Laporan Anda tetap tercatat pada riwayat."
}

func (s *service) Cancel(ctx context.Context, actor Actor, id string, input dto.CancelJobOrderDTO) (*dto.CancelResult, error) {
	orderID, err := parseID(id)
	if err != nil {
		return nil, err
	}

	var (
		outcome CancelOutcome
		fee     CancelFee
		message string
	)

	err = s.repo.Transaction(ctx, func(tx Repository) error {
		order, err := lockOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		// Batas baca berlaku lebih dulu daripada wewenang membatalkan: menjawab
		// "tidak berwenang" atau "sudah selesai" untuk order yang tak boleh
		// dilihat membocorkan keberadaan dan statusnya (asumsi A-03).
		if !VisibleTo(actor, order) {
			return apperror.NotFound("Job order tidak ditemukan")
		}
		if input.ExpectedVersion != nil {
			if err := ensureVersion(order, *input.ExpectedVersion); err != nil {
				return err
			}
		}

		decision := EvaluateCancel(actor.Role, order.CurrentStatus)
		outcome, fee, message = decision.Outcome, decision.Fee, decision.Message

		if decision.Outcome == CancelForbidden {
			return apperror.Conflict(decision.Message)
		}

		now := time.Now().UTC()

		if decision.Outcome == CancelNeedsApproval {
			// Hanya boleh ada satu permintaan pending per order. Aturan ini
			// ditegakkan di sini, bukan lewat partial unique index, karena index
			// parsial tidak bisa ditulis lewat tag GORM dan akan dianggap drift
			// lalu di-DROP pada diff Atlas berikutnya.
			if _, err := tx.FindPendingCancellation(ctx, order.ID); err == nil {
				return apperror.Conflict("Permintaan pembatalan untuk order ini sudah diajukan dan masih menunggu keputusan koordinator.")
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			request := &schema.CancellationRequest{
				JobOrderID:    order.ID,
				RequestedByID: actor.ID,
				Reason:        input.Reason,
				Status:        schema.CancellationPending,
			}
			if err := tx.CreateCancellation(ctx, request); err != nil {
				return err
			}

			// Permintaan pembatalan ikut tercatat di riwayat sebagai event yang
			// belum diterapkan. Dengan begitu setiap perubahan yang terlihat di
			// layar punya seq — kursor real-time tidak perlu tahu ada dua jenis
			// perubahan yang berbeda.
			event, err := insertRejected(ctx, tx, order, schema.StatusCancelled, actor, now,
				RejectionPendingApproval, &input.Reason)
			if err != nil {
				return err
			}
			return tx.Notify(ctx, event.Seq, order.ID)
		}

		event, err := insertTransition(ctx, tx, order, schema.StatusCancelled, actor, now, now, &input.Reason)
		if err != nil {
			return err
		}

		order.CurrentStatus = schema.StatusCancelled
		order.StatusChangedAt = now
		if err := tx.UpdateOrderStatus(ctx, order, order.Version); err != nil {
			return err
		}

		// Koordinator boleh membatalkan langsung sekalipun ada permintaan klien
		// yang masih menunggu. Permintaan itu terpenuhi, bukan menggantung.
		if _, err := holdCancellationForSettlement(ctx, tx, order, actor, now); err != nil {
			return err
		}

		return tx.Notify(ctx, event.Seq, order.ID)
	})
	if err != nil {
		return nil, err
	}

	snapshot, err := s.Snapshot(ctx, orderID)
	if err != nil {
		return nil, err
	}

	status := "cancelled"
	if outcome == CancelNeedsApproval {
		status = "pending_approval"
	}

	return &dto.CancelResult{
		Status:  status,
		Fee:     string(fee),
		Message: message,
		Order:   *snapshot,
	}, nil
}

func (s *service) DecideCancellation(ctx context.Context, actor Actor, id, requestID string, input dto.DecideCancellationDTO) (*dto.JobOrderResponse, error) {
	if actor.Role != schema.RoleAdmin {
		return nil, apperror.Forbidden("Hanya koordinator yang dapat memutuskan permintaan pembatalan")
	}

	orderID, err := parseID(id)
	if err != nil {
		return nil, err
	}
	reqID, err := parseUUIDField(requestID, "request_id")
	if err != nil {
		return nil, err
	}

	err = s.repo.Transaction(ctx, func(tx Repository) error {
		order, err := lockOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}

		request, err := tx.FindCancellationByID(ctx, reqID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.NotFound("Permintaan pembatalan tidak ditemukan")
			}
			return err
		}
		if request.JobOrderID != order.ID {
			return apperror.NotFound("Permintaan pembatalan tidak ditemukan")
		}
		if request.Status == schema.CancellationPendingSettlement {
			return apperror.Conflict(fmt.Sprintf(
				"Pekerjaan sudah %s sebelum permintaan ini sempat diputuskan, sehingga statusnya tidak dapat diubah lagi. Yang tersisa adalah keputusan penyelesaian komersialnya.",
				StatusLabel(order.CurrentStatus),
			))
		}
		if request.Status != schema.CancellationPending {
			return apperror.Conflict("Permintaan pembatalan ini sudah diputuskan sebelumnya.")
		}
		// Keputusan B-10. Jalur tulis lain sudah menutup permintaan yang gugur
		// begitu ordernya final, jadi pemeriksaan ini semestinya tak pernah
		// tercapai — ia ada karena status final tidak boleh dibuka kembali oleh
		// jalur mana pun, bukan hanya oleh jalur yang kebetulan sudah dijaga.
		if order.CurrentStatus.IsFinal() {
			return apperror.Conflict(fmt.Sprintf(
				"Pekerjaan sudah %s sebelum permintaan ini diputuskan, sehingga pembatalan tidak dapat lagi diterapkan. Selesaikan aspek komersialnya dengan klien.",
				StatusLabel(order.CurrentStatus),
			))
		}

		now := time.Now().UTC()
		request.DecidedByID = &actor.ID
		request.DecidedAt = &now
		request.DecisionNote = input.Note

		if input.Decision == "approve" {
			request.Status = schema.CancellationApproved
			if err := tx.UpdateCancellation(ctx, request); err != nil {
				return err
			}

			event, err := insertTransition(ctx, tx, order, schema.StatusCancelled, actor, now, now, input.Note)
			if err != nil {
				return err
			}

			order.CurrentStatus = schema.StatusCancelled
			order.StatusChangedAt = now
			if err := tx.UpdateOrderStatus(ctx, order, order.Version); err != nil {
				return err
			}
			return tx.Notify(ctx, event.Seq, order.ID)
		}

		request.Status = schema.CancellationRejected
		if err := tx.UpdateCancellation(ctx, request); err != nil {
			return err
		}

		// Pekerjaan berlanjut. Penolakan tetap masuk riwayat supaya klien
		// melihat keputusannya, bukan sekadar melihat permintaannya menghilang.
		event, err := insertRejected(ctx, tx, order, schema.StatusCancelled, actor, now,
			RejectionCancellationRejected, input.Note)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, event.Seq, order.ID)
	})
	if err != nil {
		return nil, err
	}

	return s.Snapshot(ctx, orderID)
}

// SettleCancellation menutup permintaan pembatalan yang tersusul selesainya
// pekerjaan, dengan mencatat siapa menanggung biayanya (keputusan B-10).
//
// Statusnya tidak ikut berubah — pekerjaannya memang benar-benar dikerjakan.
// Yang dijawab di sini adalah pertanyaan komersial yang tertinggal, dan
// jawabannya masuk riwayat supaya klien serta CS membaca hal yang sama.
func (s *service) SettleCancellation(ctx context.Context, actor Actor, id, requestID string, input dto.SettleCancellationDTO) (*dto.JobOrderResponse, error) {
	if actor.Role != schema.RoleAdmin {
		return nil, apperror.Forbidden("Hanya koordinator yang dapat memutuskan penyelesaian")
	}

	orderID, err := parseID(id)
	if err != nil {
		return nil, err
	}
	reqID, err := parseUUIDField(requestID, "request_id")
	if err != nil {
		return nil, err
	}

	err = s.repo.Transaction(ctx, func(tx Repository) error {
		order, err := lockOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}

		request, err := tx.FindCancellationByID(ctx, reqID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.NotFound("Permintaan pembatalan tidak ditemukan")
			}
			return err
		}
		if request.JobOrderID != order.ID {
			return apperror.NotFound("Permintaan pembatalan tidak ditemukan")
		}
		if request.Status == schema.CancellationSettled {
			return apperror.Conflict("Penyelesaian untuk permintaan ini sudah dicatat sebelumnya.")
		}
		if request.Status != schema.CancellationPendingSettlement {
			return apperror.Conflict("Permintaan ini belum menunggu penyelesaian. Bila pekerjaan masih berjalan, putuskan pembatalannya lebih dulu.")
		}

		now := time.Now().UTC()
		outcome := schema.SettlementOutcome(input.Outcome)

		request.Status = schema.CancellationSettled
		request.SettlementOutcome = &outcome
		request.DecidedByID = &actor.ID
		request.DecidedAt = &now
		request.DecisionNote = &input.Note
		if err := tx.UpdateCancellation(ctx, request); err != nil {
			return err
		}

		event, err := insertRejected(ctx, tx, order, schema.StatusCancelled, actor, now,
			RejectionSettlementDecided, ptr(SettlementLabel(outcome)+" — "+input.Note))
		if err != nil {
			return err
		}

		return tx.Notify(ctx, event.Seq, order.ID)
	})
	if err != nil {
		return nil, err
	}

	return s.Snapshot(ctx, orderID)
}

func (s *service) Correct(ctx context.Context, actor Actor, id string, input dto.CorrectStatusDTO) (*dto.JobOrderResponse, error) {
	if actor.Role != schema.RoleAdmin {
		return nil, apperror.Forbidden("Hanya koordinator yang dapat mengoreksi status")
	}

	orderID, err := parseID(id)
	if err != nil {
		return nil, err
	}

	err = s.repo.Transaction(ctx, func(tx Repository) error {
		order, err := lockOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if err := ensureVersion(order, input.ExpectedVersion); err != nil {
			return err
		}

		to := schema.JobStatus(input.ToStatus)
		if to == order.CurrentStatus {
			return apperror.BadRequest("Status yang dikoreksi sama dengan status saat ini")
		}

		now := time.Now().UTC()
		from := order.CurrentStatus

		// Koreksi sengaja melewati tabel transisi: justru kemampuan mundur
		// inilah alasan jalur ini ada (keputusan B-06). Yang menggantikan
		// validasi transisi adalah alasan yang wajib diisi dan jejak audit.
		event := &schema.JobStatusEvent{
			JobOrderID:   order.ID,
			FromStatus:   &from,
			ToStatus:     to,
			ActorID:      &actor.ID,
			ActorRole:    actor.Role,
			OccurredAt:   now,
			ReceivedAt:   now,
			Accepted:     true,
			IsCorrection: true,
			Reason:       &input.Reason,
		}
		if err := tx.InsertEvent(ctx, event); err != nil {
			return err
		}

		order.CurrentStatus = to
		order.StatusChangedAt = now
		if err := tx.UpdateOrderStatus(ctx, order, input.ExpectedVersion); err != nil {
			return err
		}

		notifySeq := event.Seq
		if to.IsFinal() {
			closing, err := holdCancellationForSettlement(ctx, tx, order, actor, now)
			if err != nil {
				return err
			}
			if closing != nil {
				notifySeq = closing.Seq
			}
		}

		return tx.Notify(ctx, notifySeq, order.ID)
	})
	if err != nil {
		return nil, err
	}

	return s.Snapshot(ctx, orderID)
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

func lockOrder(ctx context.Context, tx Repository, id uuid.UUID) (*schema.JobOrder, error) {
	order, err := tx.LockOrder(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("Job order tidak ditemukan")
		}
		return nil, err
	}
	return order, nil
}

// ensureVersion menegakkan keputusan B-09. Menerima kedua perubahan berarti
// salah satunya hilang tanpa ada yang menyadari; penolakan yang terlihat jauh
// lebih baik daripada kehilangan yang tidak terlihat.
func ensureVersion(order *schema.JobOrder, expected int) error {
	if order.Version == expected {
		return nil
	}
	return apperror.Conflict(fmt.Sprintf(
		"Order ini baru saja diubah orang lain, dan sekarang berstatus %s. Tampilan Anda sudah diperbarui — silakan periksa lalu ulangi bila masih diperlukan.",
		StatusLabel(order.CurrentStatus),
	))
}

// holdCancellationForSettlement menangani permintaan pembatalan yang masih
// menunggu keputusan ketika ordernya sudah mencapai status final lewat jalur
// lain. Dipanggil dari setiap jalur tulis yang bisa membuat order menjadi
// final, sehingga invariannya berlaku menyeluruh: order berstatus final tidak
// pernah menyisakan permintaan yang masih bisa mengubah statusnya.
//
// Tanpa ini, menyetujui permintaan tersebut akan memindahkan order keluar dari
// status final — padahal Completed, Failed, dan Cancelled tidak punya transisi
// keluar.
//
// Permintaan yang berakhir pada order Cancelled sudah terpenuhi dan selesai di
// sini. Selebihnya permintaan TIDAK ditutup, melainkan berpindah menunggu
// penyelesaian komersial (keputusan B-10): permintaan klien tidak cacat — ia
// masuk selagi pekerjaan berjalan, dan yang membuat pekerjaan tetap jalan
// adalah aturan perusahaan sendiri (B-05). Menutupnya sepihak berarti
// perusahaan memenangkan dirinya atas konsekuensi keputusannya sendiri, dan
// klien akan menelepon CS — persis yang hendak dihilangkan sistem ini.
func holdCancellationForSettlement(
	ctx context.Context, tx Repository, order *schema.JobOrder, actor Actor, now time.Time,
) (*schema.JobStatusEvent, error) {
	request, err := tx.FindPendingCancellation(ctx, order.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if order.CurrentStatus == schema.StatusCancelled {
		request.Status = schema.CancellationApproved
		request.DecidedAt = &now
		if actor.Role == schema.RoleAdmin {
			request.DecidedByID = &actor.ID
		}
		request.DecisionNote = ptr("Order dibatalkan koordinator, sehingga permintaan ini terpenuhi.")
		if err := tx.UpdateCancellation(ctx, request); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// DecidedAt sengaja dibiarkan kosong: belum ada yang memutuskan apa pun.
	// Yang berubah adalah pertanyaannya, dari "dibatalkan atau tidak" menjadi
	// "pekerjaan terlanjur selesai, siapa menanggung".
	request.Status = schema.CancellationPendingSettlement
	if err := tx.UpdateCancellation(ctx, request); err != nil {
		return nil, err
	}

	return insertRejected(ctx, tx, order, schema.StatusCancelled, actor, now,
		RejectionSettlementPending, ptr(fmt.Sprintf(
			"Pekerjaan sudah %s sebelum permintaan pembatalan diputuskan. Statusnya tidak berubah, dan penyelesaian komersialnya menunggu keputusan koordinator.",
			StatusLabel(order.CurrentStatus),
		)))
}

func insertTransition(
	ctx context.Context, tx Repository, order *schema.JobOrder,
	to schema.JobStatus, actor Actor, occurredAt, receivedAt time.Time, reason *string,
) (*schema.JobStatusEvent, error) {
	from := order.CurrentStatus
	event := &schema.JobStatusEvent{
		JobOrderID: order.ID,
		FromStatus: &from,
		ToStatus:   to,
		ActorID:    &actor.ID,
		ActorRole:  actor.Role,
		OccurredAt: occurredAt,
		ReceivedAt: receivedAt,
		Accepted:   true,
		Reason:     reason,
	}
	if err := tx.InsertEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func insertRejected(
	ctx context.Context, tx Repository, order *schema.JobOrder,
	to schema.JobStatus, actor Actor, at time.Time, rejection string, reason *string,
) (*schema.JobStatusEvent, error) {
	from := order.CurrentStatus
	event := &schema.JobStatusEvent{
		JobOrderID:      order.ID,
		FromStatus:      &from,
		ToStatus:        to,
		ActorID:         &actor.ID,
		ActorRole:       actor.Role,
		OccurredAt:      at,
		ReceivedAt:      at,
		Accepted:        false,
		RejectionReason: &rejection,
		Reason:          reason,
	}
	if err := tx.InsertEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func parseID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, apperror.BadRequest("Format id tidak valid").Wrap(err)
	}
	return parsed, nil
}

func parseUUIDField(value, field string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.BadRequest("Format " + field + " tidak valid").Wrap(err)
	}
	return parsed, nil
}

func ptr[T any](v T) *T { return &v }
