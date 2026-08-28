-- +goose Up
-- Keputusan B-10: permintaan pembatalan yang tersusul selesainya pekerjaan
-- tidak ditutup, melainkan menunggu keputusan penyelesaian komersial. Hasilnya
-- disimpan terpisah dari decision_note karena yang satu dapat dihitung untuk
-- laporan, sedangkan yang lain hanya dapat dibaca.
ALTER TABLE "cancellation_requests" ADD COLUMN "settlement_outcome" varchar(20);

-- Permintaan yang sempat ditandai gugur pada revisi sebelumnya dikembalikan ke
-- antrean koordinator: keputusan komersialnya memang belum pernah diambil.
UPDATE "cancellation_requests" SET "status" = 'pending_settlement', "decided_at" = NULL,
       "decided_by_id" = NULL, "decision_note" = NULL
 WHERE "status" = 'obsolete';

-- +goose Down
ALTER TABLE "cancellation_requests" DROP COLUMN "settlement_outcome";
