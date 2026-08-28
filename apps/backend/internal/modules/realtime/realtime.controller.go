package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/shared/ctxkey"
)

// keepAliveInterval menjaga koneksi tetap hidup melewati proxy yang memutus
// sambungan idle. Komentar SSE (baris diawali ":") diabaikan klien, tetapi tetap
// menghitung sebagai lalu lintas.
const keepAliveInterval = 25 * time.Second

// Controller melayani satu endpoint: stream perubahan job order.
type Controller struct {
	hub    *Hub
	orders SnapshotProvider
}

func NewController(hub *Hub, orders SnapshotProvider) *Controller {
	return &Controller{hub: hub, orders: orders}
}

// Stream godoc
//
//	@Summary		Stream perubahan job order (SSE)
//	@Description	Mengirim keadaan terkini setiap order yang berubah. Cakupan ditentukan peran pemanggil. Saat menyambung ulang, browser mengirim Last-Event-ID sendiri; pada koneksi pertama klien menyertakan last_event_id lewat query.
//	@Tags			realtime
//	@Produce		text/event-stream
//	@Param			X-Actor-Id		header	string	true	"Id pengguna yang bertindak"
//	@Param			last_event_id	query	int		false	"Kursor seq terakhir yang sudah dimiliki klien"
//	@Success		200				{string}	string	"aliran text/event-stream"
//	@Router			/stream [get]
func (ctl *Controller) Stream(c *gin.Context) {
	user, ok := ctxkey.Actor(c)
	if !ok {
		c.Error(apperror.Unauthorized("Identitas pemanggil tidak dikenali")) //nolint:errcheck
		return
	}

	scope := Scope{Role: user.Role, CompanyID: user.CompanyID, InspectorID: &user.ID}

	cursor, err := readCursor(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.Error(apperror.Internal("Koneksi ini tidak mendukung streaming")) //nolint:errcheck
		return
	}

	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	// Mencegah reverse proxy menahan potongan respons sampai buffernya penuh,
	// yang membuat stream tampak macet.
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Berlangganan LEBIH DULU, baru memutar ulang. Urutan ini disengaja:
	// perubahan yang terjadi di sela keduanya akan terkirim dua kali, dan itu
	// jauh lebih baik daripada tidak terkirim sama sekali. Klien mengabaikan
	// pesan ber-seq yang sudah diterapkan, sehingga duplikat tidak berbahaya —
	// sedangkan celah akan menghasilkan layar yang diam-diam basi.
	messages, unsubscribe := ctl.hub.Subscribe(scope)
	defer unsubscribe()

	if err := ctl.replay(c, flusher, scope, cursor); err != nil {
		return
	}

	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case msg, open := <-messages:
			if !open {
				return
			}
			if !writeEvent(c, flusher, msg.Seq, msg.Order) {
				return
			}
		case <-ticker.C:
			if _, err := fmt.Fprint(c.Writer, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// replay mengirim keadaan terkini setiap order yang berubah sejak kursor klien.
//
// Yang dikirim adalah keadaan sekarang, bukan setiap perubahan antara. Klien
// yang terputus lima menit hanya perlu tahu di mana ordernya berada sekarang;
// riwayat lengkapnya tetap tersedia lewat GET /orders/{id}/events.
func (ctl *Controller) replay(c *gin.Context, flusher http.Flusher, scope Scope, cursor int64) error {
	if cursor <= 0 {
		return nil
	}

	orders, err := ctl.orders.SnapshotsChangedSince(c.Request.Context(), cursor)
	if err != nil {
		// Koneksi sudah terlanjur dibuka dengan status 200, jadi kegagalan di
		// sini tidak bisa lagi dijadikan response error. Stream ditutup, dan
		// klien akan menyambung ulang dengan kursor yang sama.
		return err
	}

	for _, order := range orders {
		if !scope.Allows(order) {
			continue
		}
		if !writeEvent(c, flusher, order.Seq, order) {
			return errClientGone
		}
	}
	return nil
}

var errClientGone = fmt.Errorf("klien menutup koneksi")

// writeEvent menulis satu pesan SSE. Field id diisi seq — itulah yang dikirim
// balik browser sebagai Last-Event-ID ketika ia menyambung ulang sendiri,
// sehingga kursor pemulihan tidak perlu dikelola klien secara manual.
func writeEvent(c *gin.Context, flusher http.Flusher, seq int64, order dto.JobOrderResponse) bool {
	payload, err := json.Marshal(order)
	if err != nil {
		return false
	}

	if _, err := fmt.Fprintf(c.Writer, "id: %d\nevent: order.updated\ndata: %s\n\n", seq, payload); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

// readCursor membaca kursor pemulihan.
//
// Header Last-Event-ID diutamakan karena browser mengirimnya otomatis saat
// menyambung ulang. Pada koneksi PERTAMA browser tidak mengirim apa pun, jadi
// klien harus menyertakan kursornya sendiri lewat query — tanpa itu, pemulihan
// setelah halaman dimuat ulang tidak akan pernah terjadi.
func readCursor(c *gin.Context) (int64, error) {
	raw := c.GetHeader("Last-Event-ID")
	if raw == "" {
		raw = c.Query("last_event_id")
	}
	if raw == "" {
		return 0, nil
	}

	cursor, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || cursor < 0 {
		return 0, apperror.BadRequest("Kursor last_event_id harus berupa bilangan bulat tidak negatif")
	}
	return cursor, nil
}
