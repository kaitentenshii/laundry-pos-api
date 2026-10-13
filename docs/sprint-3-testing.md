# Pengujian development Sprint 3

## Kontrak baru

| Endpoint | Hak akses | Hasil |
| --- | --- | --- |
| GET /dashboard | Admin/kasir | Aktif, menunggu, dicuci, siap diambil, terlambat, penerimaan harian, pesanan dipantau |
| GET /reports/transactions?from=YYYY-MM-DD&to=YYYY-MM-DD&page=1&page_size=20 | Admin | Ringkasan semua pesanan periode dan tabel berhalaman |
| GET /settings/profile | Admin | Profil laundry |
| PUT /settings/profile | Admin | Simpan nama, contact, operating_hours, address |
| GET /laundry/identity | Admin/kasir | Nama usaha saja untuk identitas workspace |

Tanpa token menghasilkan 401; akses kasir ke laporan dan pengaturan menghasilkan 403. Filter tidak valid menghasilkan 400. Tanpa periode, laporan menggunakan awal bulan sampai hari ini menurut WIB. `to` mencakup seluruh hari terakhir.

Nilai pesanan mengikuti `orders.created_at` dan total snapshot. Penerimaan mengikuti `payments.paid_at` dan `amount_settled`, termasuk pembayaran pesanan dari periode lain. Uang diterima sebelum kembalian tidak dipakai. Sisa tagihan adalah sisa terkini dari pesanan yang masuk pada periode. Dashboard menghitung aktif sebagai RECEIVED/WASHING, siap sebagai COMPLETED, terlambat sebagai aktif dengan due_at melewati waktu snapshot. Query ringkasan dan tabel menggunakan transaksi repeatable-read.

Profil singleton dibuat oleh migrasi 000012. Data awal hanya nama Laundry Mitra; kontak/jam/alamat harus diisi admin. Nama maksimal 120 karakter, jam maksimal 120, alamat maksimal 1000. Kontak dinormalisasi menjadi nomor internasional, termasuk nomor Indonesia 08/02. Semua field wajib pada penyimpanan.

## Menjalankan pemeriksaan

Gunakan PostgreSQL QA khusus yang sudah dimigrasikan sampai versi 12. Atur `TEST_DATABASE_URL` tanpa menyimpan kredensial ke Git, lalu:

```sh
go test -count=1 ./...
go vet ./...
```

Windows juga dapat menggunakan `./scripts/test-integration.ps1`. Script mewajibkan TEST_DATABASE_URL agar pengujian PostgreSQL tidak terlewat. Fixture membuat dan membersihkan data pesanan/pembayaran/pelanggan/layanan/user; tes profil mengembalikan profil awal setelah selesai.

Pengujian baru memverifikasi batas kalender WIB, rentang tidak valid, default bulan pada pergantian tanggal UTC, pagination dengan ringkasan seluruh periode, pembayaran di periode berbeda, kembalian tunai, snapshot harga, profil singleton/persistence, serta RBAC. Pengujian order/payment/status sebelumnya mencakup konkurensi dan rollback saat pencatatan riwayat gagal.

Browser E2E tersedia pada repositori frontend melalui `npm run test:e2e`. Hasil lokal telah memverifikasi perjalanan kasir lengkap dengan API dan PostgreSQL nyata.

WhatsApp/queue/retry/webhook tetap manual. Deployment di-hold. UAT memerlukan pelaksanaan dan penerimaan mitra; checklist bukan bukti penerimaan.
