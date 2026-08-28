package realtime

import (
	"sync"

	"github.com/google/uuid"

	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/schema"
)

// Message adalah satu perubahan yang disiarkan ke klien.
//
// Yang dikirim adalah keadaan order saat ini, bukan selisihnya. Layar hanya
// perlu tahu keadaan sekarang, dan snapshot membuat klien yang baru menyambung
// dan klien yang sudah lama tersambung menerima bentuk pesan yang sama persis.
// Riwayat lengkapnya tetap tersedia lewat endpoint events.
type Message struct {
	Seq   int64
	Order dto.JobOrderResponse
}

// Scope membatasi order mana yang boleh diterima satu langganan.
type Scope struct {
	Role        schema.Role
	CompanyID   *uuid.UUID
	InspectorID *uuid.UUID
}

// Allows menyaring di server sebelum menulis ke koneksi. Penyaringan tidak
// boleh diserahkan ke klien: kerahasiaan komersial antar klien (asumsi A-03)
// harus tetap berlaku walaupun seseorang membuka stream secara langsung.
func (s Scope) Allows(order dto.JobOrderResponse) bool {
	switch s.Role {
	case schema.RoleClient:
		return s.CompanyID != nil && order.CompanyID == s.CompanyID.String()
	case schema.RoleInspector:
		return s.InspectorID != nil &&
			order.InspectorID != nil &&
			*order.InspectorID == s.InspectorID.String()
	default:
		// Koordinator dan CS memantau seluruh order berjalan.
		return true
	}
}

// subscriber adalah satu koneksi SSE yang sedang terbuka.
type subscriber struct {
	scope Scope
	ch    chan Message
}

// buffer memberi ruang bagi lonjakan pendek tanpa memblokir penyiar. Ukurannya
// kecil dengan sengaja — lihat alasannya di Broadcast.
const buffer = 16

// Hub menyiarkan pesan ke seluruh langganan di dalam satu proses.
//
// Fan-out lintas instance tidak terjadi di sini melainkan di listener: setiap
// pod menerima NOTIFY yang sama dari Postgres, lalu meneruskannya ke Hub
// miliknya sendiri. Karena itu Hub tidak perlu tahu apa pun tentang instance lain.
type Hub struct {
	mu   sync.RWMutex
	subs map[*subscriber]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[*subscriber]struct{})}
}

// Subscribe mendaftarkan langganan baru dan mengembalikan kanal pesannya
// beserta fungsi pembatalannya.
func (h *Hub) Subscribe(scope Scope) (<-chan Message, func()) {
	sub := &subscriber{scope: scope, ch: make(chan Message, buffer)}

	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()

	return sub.ch, func() { h.remove(sub) }
}

func (h *Hub) remove(sub *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.subs[sub]; !exists {
		return
	}
	delete(h.subs, sub)
	close(sub.ch)
}

// Broadcast meneruskan pesan ke seluruh langganan yang berhak menerimanya.
//
// WARNING: langganan yang kanalnya penuh DILEWATI, bukan ditunggu. Menunggu
// berarti satu klien lambat bisa menghentikan penyiaran ke semua klien lain,
// dan pada akhirnya menahan listener. Melewatinya aman karena klien membawa
// kursor: begitu ia menyusul atau menyambung ulang, perubahan yang terlewat
// dikirim ulang lewat replay. Kehilangan pesan di sini tidak berarti kehilangan
// data — hanya menunda kedatangannya.
func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for sub := range h.subs {
		if !sub.scope.Allows(msg.Order) {
			continue
		}
		select {
		case sub.ch <- msg:
		default:
		}
	}
}

// Subscribers melaporkan jumlah langganan yang sedang terbuka. Dipakai log dan
// pemeriksaan saat pengembangan.
func (h *Hub) Subscribers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
