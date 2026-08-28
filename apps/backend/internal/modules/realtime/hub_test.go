package realtime_test

import (
	"testing"

	"github.com/google/uuid"

	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/modules/realtime"
	"verifield-be/internal/schema"
)

func orderOf(companyID uuid.UUID, inspectorID *uuid.UUID) dto.JobOrderResponse {
	order := dto.JobOrderResponse{CompanyID: companyID.String()}
	if inspectorID != nil {
		id := inspectorID.String()
		order.InspectorID = &id
	}
	return order
}

func TestScopeMenyaringPerPeran(t *testing.T) {
	perusahaan := uuid.New()
	perusahaanLain := uuid.New()
	inspector := uuid.New()
	inspektorLain := uuid.New()

	milikKlien := orderOf(perusahaan, &inspector)
	milikOrangLain := orderOf(perusahaanLain, &inspektorLain)

	cases := []struct {
		name  string
		scope realtime.Scope
		order dto.JobOrderResponse
		want  bool
	}{
		{
			"klien menerima order perusahaannya",
			realtime.Scope{Role: schema.RoleClient, CompanyID: &perusahaan},
			milikKlien, true,
		},
		{
			"klien tidak menerima order perusahaan lain",
			realtime.Scope{Role: schema.RoleClient, CompanyID: &perusahaan},
			milikOrangLain, false,
		},
		{
			"klien tanpa perusahaan tidak menerima apa pun",
			realtime.Scope{Role: schema.RoleClient},
			milikKlien, false,
		},
		{
			"inspektor menerima penugasannya sendiri",
			realtime.Scope{Role: schema.RoleInspector, InspectorID: &inspector},
			milikKlien, true,
		},
		{
			"inspektor tidak menerima penugasan orang lain",
			realtime.Scope{Role: schema.RoleInspector, InspectorID: &inspector},
			milikOrangLain, false,
		},
		{
			"inspektor tidak menerima order yang belum ditugaskan",
			realtime.Scope{Role: schema.RoleInspector, InspectorID: &inspector},
			orderOf(perusahaan, nil), false,
		},
		{
			"koordinator memantau seluruh order",
			realtime.Scope{Role: schema.RoleAdmin},
			milikOrangLain, true,
		},
		{
			"cs ikut memantau seluruh order",
			realtime.Scope{Role: schema.RoleCS},
			milikOrangLain, true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.scope.Allows(c.order); got != c.want {
				t.Errorf("Allows = %v, mau %v", got, c.want)
			}
		})
	}
}

func TestBroadcastHanyaKeLanggananYangBerhak(t *testing.T) {
	perusahaanA, perusahaanB := uuid.New(), uuid.New()
	hub := realtime.NewHub()

	klienA, batalA := hub.Subscribe(realtime.Scope{Role: schema.RoleClient, CompanyID: &perusahaanA})
	klienB, batalB := hub.Subscribe(realtime.Scope{Role: schema.RoleClient, CompanyID: &perusahaanB})
	ops, batalOps := hub.Subscribe(realtime.Scope{Role: schema.RoleAdmin})
	defer batalA()
	defer batalB()
	defer batalOps()

	hub.Broadcast(realtime.Message{Seq: 7, Order: orderOf(perusahaanA, nil)})

	if msg := <-klienA; msg.Seq != 7 {
		t.Errorf("klien A menerima seq %d, mau 7", msg.Seq)
	}
	if msg := <-ops; msg.Seq != 7 {
		t.Errorf("koordinator menerima seq %d, mau 7", msg.Seq)
	}
	select {
	case msg := <-klienB:
		t.Errorf("klien B tidak boleh menerima order perusahaan lain, dapat seq %d", msg.Seq)
	default:
	}
}

func TestLanggananLambatDilewatiBukanMemblokir(t *testing.T) {
	// Menunggu klien lambat berarti satu koneksi macet bisa menghentikan
	// penyiaran ke semua koneksi lain. Klien yang terlewat memulihkan diri
	// lewat kursor saat menyambung ulang.
	hub := realtime.NewHub()
	perusahaan := uuid.New()

	lambat, batal := hub.Subscribe(realtime.Scope{Role: schema.RoleAdmin})
	defer batal()

	selesai := make(chan struct{})
	go func() {
		for i := range 200 {
			hub.Broadcast(realtime.Message{Seq: int64(i), Order: orderOf(perusahaan, nil)})
		}
		close(selesai)
	}()

	<-selesai // tidak boleh menggantung walau kanal langganan sudah penuh

	terbaca := 0
	for {
		select {
		case <-lambat:
			terbaca++
			continue
		default:
		}
		break
	}

	if terbaca == 0 {
		t.Error("langganan lambat seharusnya tetap menerima sebagian pesan")
	}
	if terbaca == 200 {
		t.Error("kanal berbuffer tidak mungkin menampung seluruh pesan tanpa dibaca")
	}
}

func TestUnsubscribeMenutupKanal(t *testing.T) {
	hub := realtime.NewHub()
	ch, batal := hub.Subscribe(realtime.Scope{Role: schema.RoleAdmin})

	if hub.Subscribers() != 1 {
		t.Fatalf("jumlah langganan = %d, mau 1", hub.Subscribers())
	}

	batal()
	if _, terbuka := <-ch; terbuka {
		t.Error("kanal harus tertutup setelah langganan dibatalkan")
	}
	if hub.Subscribers() != 0 {
		t.Errorf("jumlah langganan = %d, mau 0", hub.Subscribers())
	}

	batal() // pembatalan berulang tidak boleh panik karena menutup kanal dua kali
}
