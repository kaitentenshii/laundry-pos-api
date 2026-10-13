# Checklist UAT Sprint 3

Status: **belum dilaksanakan oleh mitra**. Pengujian development otomatis sudah dilakukan; penerimaan mitra harus dicatat terpisah.

Isi nama penguji, tanggal, perangkat/browser, viewport, dan versi commit frontend/backend sebelum mulai. Gunakan akun admin dan kasir QA, serta data uji yang disepakati.

| No | Skenario dan kriteria penerimaan | Hasil mitra / catatan |
| --- | --- | --- |
| 1 | Login admin/kasir; password salah menunjukkan error; logout menutup akses | Belum diuji |
| 2 | Kasir tidak melihat menu admin; URL laporan/pengaturan langsung ditolak | Belum diuji |
| 3 | Cari pelanggan nama/nomor, pilih, ganti tanpa kehilangan draft | Belum diuji |
| 4 | Tambah pelanggan dari pesanan dengan alamat/catatan opsional; tersimpan dan terpilih | Belum diuji |
| 5 | Pesanan 4 kg × Rp8.000 + 1 pcs × Rp25.000 menghasilkan Rp57.000 | Belum diuji |
| 6 | Ubah harga layanan; pesanan lama tetap menggunakan harga saat transaksi | Belum diuji |
| 7 | Batal konfirmasi status tidak mengubah pesanan; dicuci dan selesai mencatat aktor/waktu | Belum diuji |
| 8 | Pesanan selesai belum lunas tidak dapat diambil; bayar tunai Rp60.000, kembali Rp3.000 | Belum diuji |
| 9 | Konfirmasi pengambilan setelah lunas; klik ulang tidak menggandakan riwayat | Belum diuji |
| 10 | Dashboard sesuai data aktif/siap/terlambat; penerimaan bertambah Rp57.000, bukan Rp60.000 | Belum diuji |
| 11 | Laporan apply/reset periode, tanggal akhir inklusif WIB, empty period, angka dan tabel konsisten | Belum diuji |
| 12 | Nama/kontak/jam/alamat profil tersimpan setelah refresh; sidebar berubah setelah save | Belum diuji |
| 13 | Nomor kontak tidak valid ditolak; isian lain tidak hilang; error koneksi dapat dicoba ulang | Belum diuji |
| 14 | Tampilan dan kontrol dapat digunakan pada perangkat mitra; catat ukuran layar | Belum diuji |
| 15 | Notifikasi WhatsApp/consent/preferences/provider delivery/retry diuji terpisah setelah implementasi manual | Menunggu manual |

Catat setiap temuan dengan langkah reproduksi, hasil yang diharapkan/aktual, prioritas, screenshot tanpa kredensial, issue GitHub, dan hasil uji ulang.

Penerimaan: nama mitra ___, tanggal ___, hasil diterima/belum ___, pengecualian ___, issue tersisa ___. Jangan menutup issue UAT sebelum bagian ini terisi berdasarkan hasil mitra.
