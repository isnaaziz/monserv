# monserv — Web Monitor Server (Golang)

Aplikasi monitoring server ringan berbasis Go untuk memantau 4+ server sekaligus: penggunaan Memory dan Disk per mount, proses yang paling banyak memakai RAM, UI web sederhana, dan alert broadcast ke Email/Slack/Telegram saat melewati ambang batas.

## Arsitektur singkat

- Mode tanpa agent (default di contoh ini): Server Pusat terhubung via SSH ke server-target dan mengambil metrik langsung (Linux).
- Mode dengan agent (opsional): Agent (biner `agent`) dipasang di tiap server, expose `GET /metrics` (JSON); Server Pusat polling endpoint tersebut.

## Fitur

- Memory: total, used, free, used%.
- Disk: per-mount total/used/free/used%.
- Top proses berdasarkan penggunaan RAM (RSS).
- Alert otomatis (ALERT & RECOVERED) dengan cooldown agar tidak spam.
- Integrasi notifikasi: Email (SMTP), Slack Webhook, Telegram Bot (opsional; isi ENV untuk mengaktifkan).

## Build

Pastikan Go terinstal. Lalu:

```sh
# dari root repo
go build -o bin/agent ./cmd/agent
go build -o bin/server ./cmd/server
```

## Menjalankan Agent di tiap server

Agent membaca port dari `AGENT_PORT` (default 9123).

```sh
AGENT_PORT=9123 ./bin/agent
```

Sistemd unit contoh (opsional):

```
[Unit]
Description=monserv agent
After=network.target

[Service]
ExecStart=/opt/monserv/bin/agent
Environment=AGENT_PORT=9123
Restart=always
User=monserv
Group=monserv

[Install]
WantedBy=multi-user.target
```

Pastikan port agent dapat diakses dari Server Pusat (firewall/security group).

## Menjalankan Server Pusat (Web UI + Alert)

Set variabel lingkungan berikut:

- `SERVERS`: daftar target, pisahkan dengan koma. Bisa campur:
  - HTTP agent: `http://10.0.0.11:9123`
  - SSH tanpa agent: `ssh://user:pass@192.168.4.3:2222`
  - Contoh SSH (sesuai kebutuhan Anda): `ssh://scada:reatuuav@192.168.4.3:2222,ssh://scada:reatuuav@192.168.4.4:2222,ssh://scada:reatuuav@192.168.4.5:2222,ssh://scada:reatuuav@192.168.4.6:2222`
- `POLL_INTERVAL_SECONDS` (opsional, default 5): interval polling.
- `MEM_THRESHOLD_PERCENT` (opsional, default 90): ambang batas alert memory (%).
- `DISK_THRESHOLD_PERCENT` (opsional, default 90): ambang batas alert disk (%).
- `PROC_RAM_THRESHOLD_PERCENT` (opsional, default 20): ambang batas alert per-proses berdasarkan % RAM.
- `SERVER_PORT` (opsional, default 8080): port web UI.

Notifikasi (opsional; isi salah satu/lebih):

- Email (SMTP): `EMAIL_SMTP_HOST`, `EMAIL_SMTP_PORT` (default 587), `EMAIL_FROM`, `EMAIL_PASSWORD`, `EMAIL_TO` (comma-separated)
- Slack: `SLACK_WEBHOOK_URL`
- Telegram: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`

Jalankan:

```sh
# Contoh SSH tanpa agent
SERVERS="ssh://scada:reatuuav@192.168.4.3:2222,ssh://scada:reatuuav@192.168.4.4:2222,ssh://scada:reatuuav@192.168.4.5:2222,ssh://scada:reatuuav@192.168.4.6:2222" \
PROC_RAM_THRESHOLD_PERCENT=20 \
MEM_THRESHOLD_PERCENT=90 DISK_THRESHOLD_PERCENT=90 \
SERVER_PORT=8080 ./bin/server

# Atau mode agent HTTP
# SERVERS="http://10.0.0.11:9123,http://10.0.0.12:9123" ./bin/server
```

Buka: http://localhost:8080

## Catatan

- Agent menggunakan [gopsutil] untuk membaca statistik OS. Hak akses mungkin diperlukan untuk sebagian informasi pada OS tertentu.
- “Detail yang memakan disk” di level direktori bisa sangat mahal secara I/O. Untuk saat ini dashboard fokus ke penggunaan disk per-mount. Jika ingin top direktori, bisa ditambahkan worker yang menjalankan `du` terjadwal pada path tertentu dan mengekspose hasilnya via agent (lanjutan).
- Alert memiliki cooldown 30 menit; alert “RECOVERED” terkirim saat nilai turun di bawah threshold.

## Pengembangan

Struktur penting:

- `cmd/agent`: HTTP agent (`/metrics`, `/health`)
- `cmd/server`: web UI + poller + alert
- `internal/agent`: pengumpul metrik lokal
- `internal/server`: konfigurasi, poller, state
- `internal/notifier`: integrasi Email/Slack/Telegram
- `web/`: template dan assets UI

Lisensi: MIT

[gopsutil]: https://github.com/shirou/gopsutil
