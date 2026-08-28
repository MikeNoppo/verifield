/** Memastikan tipe yang ter-commit masih sesuai dengan anotasi Go.
 *
 *  Inilah penjaga sesungguhnya dari paket ini. Tipe yang di-generate hanya
 *  berguna kalau ada yang menangkap saat ia tertinggal — tanpa pemeriksaan ini,
 *  backend bisa berubah dan frontend tetap lolos kompilasi memakai tipe lama. */
import { $ } from "bun"

const SCHEMA = "./src/schema.ts"
const SPEC = "./openapi.json"

const sebelum = {
  spec: await Bun.file(SPEC).text(),
  schema: await Bun.file(SCHEMA).text(),
}

await $`bun run generate`.quiet()

const sesudah = {
  spec: await Bun.file(SPEC).text(),
  schema: await Bun.file(SCHEMA).text(),
}

const menyimpang = [
  sebelum.spec !== sesudah.spec ? SPEC : null,
  sebelum.schema !== sesudah.schema ? SCHEMA : null,
].filter(Boolean)

if (menyimpang.length > 0) {
  console.error(
    `Kontrak sudah tertinggal dari anotasi Go: ${menyimpang.join(", ")}\n` +
      `Jalankan "bun run contract:generate" lalu commit hasilnya.`,
  )
  process.exit(1)
}

console.log("Kontrak selaras dengan anotasi Go.")
