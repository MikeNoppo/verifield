package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/schema"
	"verifield-be/internal/shared/ctxkey"
)

// HeaderActorID adalah header yang menyatakan siapa yang sedang melakukan aksi.
const HeaderActorID = "X-Actor-Id"

// UserLoader adalah bagian modul user yang dibutuhkan middleware ini.
type UserLoader interface {
	FindEntityByID(ctx context.Context, id string) (*schema.User, error)
}

// RequireActor mengganti autentikasi selama autentikasi berada di luar cakupan
// PoC (dokumen konteks bisnis bagian 13). Identitas diambil dari header
// X-Actor-Id, lalu dimuat menjadi user sungguhan sehingga peran dan perusahaan
// pemiliknya tetap ditegakkan di server.
//
// WARNING: siapa pun yang bisa menebak sebuah UUID dapat bertindak sebagai
// pemiliknya. Ini dapat diterima karena instance PoC tidak boleh menyentuh
// jaringan publik. Menggantinya nanti hanya menyentuh satu berkas ini: middleware
// mengisi ctxkey.SetActor dari klaim JWT, dan seluruh service tidak berubah
// karena mereka sudah menerima aktor sebagai parameter, bukan menggalinya sendiri.
func RequireActor(users UserLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderActorID)
		if id == "" {
			// EventSource di browser tidak bisa memasang header sama sekali,
			// sehingga stream SSE tidak punya cara lain menyebutkan identitas.
			//
			// WARNING: identitas di query string ikut tercatat di log akses dan
			// riwayat peramban. Pada sistem sungguhan ini ditangani cookie
			// sesi — yang justru terkirim otomatis oleh EventSource. Jalur ini
			// ada karena PoC memang tidak punya sesi sama sekali.
			id = c.Query("actor_id")
		}
		if id == "" {
			c.Error(apperror.Unauthorized( //nolint:errcheck
				"Header " + HeaderActorID + " wajib diisi. Autentikasi berada di luar cakupan PoC, sehingga peran dipilih di frontend.",
			))
			c.Abort()
			return
		}

		user, err := users.FindEntityByID(c.Request.Context(), id)
		if err != nil {
			c.Error(err) //nolint:errcheck
			c.Abort()
			return
		}
		if !user.IsActive {
			c.Error(apperror.Forbidden("Akun ini sedang tidak aktif")) //nolint:errcheck
			c.Abort()
			return
		}

		ctxkey.SetActor(c, user)
		c.Next()
	}
}
