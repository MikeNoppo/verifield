package joborder_test

import (
	"testing"

	"github.com/google/uuid"

	"verifield-be/internal/modules/joborder"
	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/schema"
)

// TestVisibleTo menegakkan batas baca per peran: klien hanya order
// perusahaannya (A-03), inspektor hanya order yang ditugaskan kepadanya,
// koordinator dan CS melihat semuanya.
func TestVisibleTo(t *testing.T) {
	perusahaanA := uuid.New()
	perusahaanB := uuid.New()
	inspektorA := uuid.New()
	inspektorB := uuid.New()
	klienA := uuid.New()

	orderSEN := &schema.JobOrder{
		CompanyID:   perusahaanA,
		InspectorID: &inspektorA,
	}

	cases := []struct {
		name    string
		actor   joborder.Actor
		order   *schema.JobOrder
		visible bool
	}{
		{
			"klien melihat order perusahaannya",
			joborder.Actor{ID: klienA, Role: schema.RoleClient, CompanyID: &perusahaanA},
			orderSEN, true,
		},
		{
			"klien tidak melihat order perusahaan lain",
			joborder.Actor{ID: klienA, Role: schema.RoleClient, CompanyID: &perusahaanB},
			orderSEN, false,
		},
		{
			"klien tanpa perusahaan tidak melihat apa pun",
			joborder.Actor{ID: klienA, Role: schema.RoleClient},
			orderSEN, false,
		},
		{
			"inspektor melihat order yang ditugaskan kepadanya",
			joborder.Actor{ID: inspektorA, Role: schema.RoleInspector},
			orderSEN, true,
		},
		{
			"inspektor tidak melihat order inspektor lain",
			joborder.Actor{ID: inspektorB, Role: schema.RoleInspector},
			orderSEN, false,
		},
		{
			"inspektor tidak melihat order yang belum ditugaskan",
			joborder.Actor{ID: inspektorA, Role: schema.RoleInspector},
			&schema.JobOrder{CompanyID: perusahaanA}, false,
		},
		{
			"koordinator melihat semua order",
			joborder.Actor{ID: klienA, Role: schema.RoleAdmin},
			orderSEN, true,
		},
		{
			"cs melihat semua order",
			joborder.Actor{ID: klienA, Role: schema.RoleCS},
			orderSEN, true,
		},
	}

	for _, c := range cases {
		if got := joborder.VisibleTo(c.actor, c.order); got != c.visible {
			t.Errorf("%s: VisibleTo = %v, ingin %v", c.name, got, c.visible)
		}
	}
}

// TestScopeQuery menegakkan bahwa saringan daftar ditimpa server, bukan
// dipercayakan pada query string yang dikirim pemanggil.
func TestScopeQuery(t *testing.T) {
	perusahaan := uuid.New()
	inspector := uuid.New()
	orangLain := uuid.New()

	t.Run("inspektor tidak bisa melihat daftar inspektor lain", func(t *testing.T) {
		actor := joborder.Actor{ID: inspector, Role: schema.RoleInspector}
		got, err := joborder.ScopeQuery(actor, dto.ListQuery{InspectorID: orangLain.String()})
		if err != nil {
			t.Fatalf("ScopeQuery: %v", err)
		}
		if got.InspectorID != inspector.String() {
			t.Errorf("InspectorID = %q, ingin %q", got.InspectorID, inspector.String())
		}
	})

	t.Run("klien tidak bisa melihat daftar perusahaan lain", func(t *testing.T) {
		actor := joborder.Actor{ID: uuid.New(), Role: schema.RoleClient, CompanyID: &perusahaan}
		got, err := joborder.ScopeQuery(actor, dto.ListQuery{CompanyID: uuid.NewString()})
		if err != nil {
			t.Fatalf("ScopeQuery: %v", err)
		}
		if got.CompanyID != perusahaan.String() {
			t.Errorf("CompanyID = %q, ingin %q", got.CompanyID, perusahaan.String())
		}
	})

	t.Run("klien tanpa perusahaan ditolak", func(t *testing.T) {
		actor := joborder.Actor{ID: uuid.New(), Role: schema.RoleClient}
		if _, err := joborder.ScopeQuery(actor, dto.ListQuery{}); err == nil {
			t.Error("ScopeQuery = nil, ingin error")
		}
	})

	t.Run("koordinator boleh menyaring bebas", func(t *testing.T) {
		actor := joborder.Actor{ID: uuid.New(), Role: schema.RoleAdmin}
		got, err := joborder.ScopeQuery(actor, dto.ListQuery{InspectorID: orangLain.String()})
		if err != nil {
			t.Fatalf("ScopeQuery: %v", err)
		}
		if got.InspectorID != orangLain.String() {
			t.Errorf("InspectorID = %q, ingin tetap %q", got.InspectorID, orangLain.String())
		}
	})
}
