package realtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"verifield-be/internal/modules/joborder"
	"verifield-be/internal/modules/joborder/dto"
)

// SnapshotProvider adalah bagian modul joborder yang dibutuhkan lapisan ini:
// menyusun keadaan terkini satu order, dan memutar ulang order yang berubah
// sejak sebuah kursor.
type SnapshotProvider interface {
	Snapshot(ctx context.Context, orderID uuid.UUID) (*dto.JobOrderResponse, error)
	SnapshotsChangedSince(ctx context.Context, seq int64) ([]dto.JobOrderResponse, error)
}

const (
	reconnectDelay = 2 * time.Second
	maxBackoff     = 30 * time.Second
)

// Listener menerjemahkan NOTIFY dari Postgres menjadi siaran ke Hub lokal.
//
// Inilah yang menjawab "bagaimana event dari instance A sampai ke klien yang
// terhubung ke instance B": tidak ada instance yang berbicara langsung ke
// instance lain. Semuanya mendengarkan kanal Postgres yang sama, dan Postgres
// sudah menjadi dependensi bersama — sehingga fan-out lintas pod tidak menambah
// satu pun komponen infrastruktur baru.
type Listener struct {
	dsn    string
	hub    *Hub
	orders SnapshotProvider
	log    *slog.Logger
}

func NewListener(dsn string, hub *Hub, orders SnapshotProvider, log *slog.Logger) *Listener {
	return &Listener{dsn: dsn, hub: hub, orders: orders, log: log}
}

// Run mendengarkan sampai ctx dibatalkan, menyambung ulang bila koneksi putus.
//
// Sambungan yang putus berarti ada jendela waktu tanpa siaran. Itu dapat
// diterima karena klien membawa kursor: begitu ia menyambung ulang, seluruh
// perubahan yang terlewat dikirim ulang lewat replay. Yang hilang saat listener
// putus hanyalah kecepatan, bukan data.
func (l *Listener) Run(ctx context.Context) {
	backoff := reconnectDelay

	for ctx.Err() == nil {
		err := l.listen(ctx)
		if ctx.Err() != nil {
			return
		}

		l.log.Error("listener perubahan terputus, menyambung ulang",
			"error", err, "jeda", backoff)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}

		if backoff < maxBackoff {
			backoff *= 2
		}
	}
}

func (l *Listener) listen(ctx context.Context) error {
	// WARNING: koneksi ini berdiri sendiri, di luar connection pool GORM.
	// Sesi yang sedang LISTEN memegang koneksinya sepanjang waktu, sehingga
	// meminjamnya dari pool akan menahan satu slot selamanya.
	conn, err := pgx.Connect(ctx, l.dsn)
	if err != nil {
		return fmt.Errorf("membuka koneksi listener: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	if _, err := conn.Exec(ctx, "LISTEN "+joborder.NotifyChannel); err != nil {
		return fmt.Errorf("LISTEN %s: %w", joborder.NotifyChannel, err)
	}

	l.log.Info("mendengarkan perubahan job order", "channel", joborder.NotifyChannel)

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("menunggu notifikasi: %w", err)
		}
		l.dispatch(ctx, notification.Payload)
	}
}

// dispatch memuat keadaan terkini order lalu menyiarkannya.
//
// Snapshot diambil sekali di sini, bukan sekali per koneksi SSE: berapa pun
// jumlah klien yang sedang terhubung, satu perubahan tetap menghasilkan satu
// query.
func (l *Listener) dispatch(ctx context.Context, payload string) {
	seq, orderID, err := parsePayload(payload)
	if err != nil {
		l.log.Warn("payload notifikasi tidak dikenali", "payload", payload, "error", err)
		return
	}

	order, err := l.orders.Snapshot(ctx, orderID)
	if err != nil {
		l.log.Error("gagal memuat order untuk disiarkan",
			"order_id", orderID, "seq", seq, "error", err)
		return
	}

	l.hub.Broadcast(Message{Seq: seq, Order: *order})
}

// parsePayload membaca format "<seq>:<order_id>" yang ditulis pg_notify.
func parsePayload(payload string) (int64, uuid.UUID, error) {
	rawSeq, rawID, found := strings.Cut(payload, ":")
	if !found {
		return 0, uuid.Nil, errors.New("format harus <seq>:<order_id>")
	}

	seq, err := strconv.ParseInt(rawSeq, 10, 64)
	if err != nil {
		return 0, uuid.Nil, fmt.Errorf("seq: %w", err)
	}

	orderID, err := uuid.Parse(rawID)
	if err != nil {
		return 0, uuid.Nil, fmt.Errorf("order_id: %w", err)
	}

	return seq, orderID, nil
}
