import { PermintaanBaruForm } from "@/components/verifield/permintaan-baru-form"
import { listInspectionTypes } from "@/lib/api/orders"

export default async function PermintaanBaru() {
  const types = await listInspectionTypes()
  return <PermintaanBaruForm types={types} />
}
