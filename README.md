# 🌐 VPS Rental — Sewa VPS/RDP Instan

Platform penyewaan VPS/RDP instan dengan pembayaran otomatis, integrasi Proxmox VE, dan panel admin lengkap. Terinspirasi dari [ecer.biz.id](https://ecer.biz.id).

Dibangun dengan **Go (Golang)** + **SQLite** + **Tailwind CSS** — single binary, zero dependency, deploy di mana saja.

---

## ✨ Fitur Utama

### 🛒 Landing Page Publik
- **Hero section** dengan dark emerald gradient (ala ecer.biz.id)
- **Feature cards**: Desktop Remote, Keamanan, Koneksi, Aktivasi Instan
- **How It Works** — 3 langkah: Pilih Paket → Bayar → Aktif
- **Order form interaktif**:
  - Pilih kategori (VPS Linux / RDP Windows / Radio Streaming)
  - Pilih paket (Cupang, Gurame, Arwana, dll)
  - Pilih durasi (1 jam → 1 bulan, makin lama makin murah)
  - Input kupon diskon (real-time validation)
  - Total harga otomatis terhitung
  - Submit → order ID + instruksi pembayaran
- **Responsive** mobile + desktop
- Font **Outfit** + **Tailwind CSS** (via CDN)

### 🛡️ Admin Panel
- **📊 Dashboard** — statistik real-time (total orders, active, pending, revenue, packages, coupons)
- **📦 Paket** — CRUD paket VPS/RDP, set specs (cores, RAM, disk) & tiers harga
- **🧾 Pesanan** — list semua order, update status (pending → paid → active → expired)
- **🎫 Kupon** — CRUD kupon diskon (persentase atau nominal tetap)
- **🖥️ Proxmox** — status koneksi, list VM/LXC, Start/Stop/Delete, Create container baru
- **⚙️ Settings** — brand name, WhatsApp, email, Proxmox connection config

### 🔌 Proxmox VE Integration
- Auto-authenticate ke Proxmox VE API (ticket-based)
- List semua LXC + QEMU VMs
- Create LXC container (auto VMID, password, specs dari paket)
- Start / Stop / Delete VM
- Monitor status node Proxmox

---

## 🏗️ Arsitektur

```
vps-rental/
├── main.go              # Entry point + embedded HTML templates + routing
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── models/
│   └── models.go        # Data models: Category, Package, Tier, Order, Coupon, Settings
├── store/
│   └── store.go         # SQLite data store + migrations + seed data
├── api/
│   └── handler.go       # HTTP API handlers (public + admin)
├── proxmox/
│   └── client.go        # Proxmox VE API client
├── static/
│   ├── css/app.css      # Custom styles
│   └── placeholder.txt   # For embed.FS
├── .gitignore
└── README.md
```

### Tech Stack
| Komponen | Teknologi |
|---|---|
| Backend | Go 1.18+ |
| Database | SQLite 3 (WAL mode) |
| Frontend | HTML + Tailwind CSS (CDN) + Vanilla JS |
| Auth | bcrypt password hashing + session token cookie |
| Virtualisasi | Proxmox VE API (LXC + QEMU) |
| Embed | `embed.FS` — binary includes all static assets |

---

## 📡 API Endpoints

### Public API (v1)

| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/categories` | Daftar semua kategori paket |
| `GET` | `/api/v1/packages` | Daftar semua paket + tiers harga |
| `POST` | `/api/v1/orders` | Buat pesanan baru |
| `POST` | `/api/v1/coupons/validate` | Validasi kupon & hitung diskon |
| `GET` | `/api/v1/track/{id}` | Lacak status pesanan |

### Admin API (memerlukan auth)

| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/admin/api/login` | Login admin → dapat token |
| `POST` | `/admin/api/logout` | Logout |
| `GET` | `/admin/api/stats` | Dashboard statistik |
| `GET/POST` | `/admin/api/packages` | List / create paket |
| `GET/PUT/DELETE` | `/admin/api/packages/{id}` | Get / update / delete paket |
| `GET/POST` | `/admin/api/categories` | List / create kategori |
| `GET` | `/admin/api/orders` | List semua pesanan |
| `GET/PUT` | `/admin/api/orders/{id}` | Get / update status pesanan |
| `GET/POST` | `/admin/api/coupons` | List / create kupon |
| `GET/PUT/DELETE` | `/admin/api/coupons/{id}` | Get / update / delete kupon |
| `GET/POST` | `/admin/api/settings` | Get / update settings |
| `GET` | `/admin/api/proxmox/status` | Status koneksi Proxmox |
| `GET` | `/admin/api/proxmox/vms` | List semua VM/LXC |
| `POST` | `/admin/api/proxmox/create` | Create LXC container baru |
| `POST` | `/admin/api/proxmox/action/{action}/{vmid}` | Start/stop/delete VM |

### Contoh Request

**Buat pesanan:**
```bash
curl -X POST http://localhost:8201/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@example.com",
    "whatsapp": "+62812345678",
    "packageId": "pkg-2",
    "durationHours": 24,
    "couponCode": "HEMAT10"
  }'
```

**Response:**
```json
{
  "id": "a1b2c3d4e5f6",
  "status": "pending",
  "totalPrice": 1080,
  "package": { "name": "Paket Gurame", ... },
  "expiresAt": "2026-10-02T04:00:00Z"
}
```

**Login admin:**
```bash
curl -X POST http://localhost:8201/admin/api/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
```

**Response:**
```json
{
  "role": "admin",
  "token": "aca63b..."
}
```

---

## 🚀 Cara Menjalankan

### Dari Source

```bash
# 1. Clone
git clone https://github.com/hasan/vps-rental.git
cd vps-rental

# 2. Build
go build -o vps-rental .

# 3. Run
PORT=8201 DB_PATH=data/vps-rental.db ./vps-rental
```

### Dari Binary

```bash
mkdir -p data
PORT=8201 DB_PATH=data/vps-rental.db ./vps-rental
```

### Environment Variables

| Variable | Default | Deskripsi |
|---|---|---|
| `PORT` | `8080` | Port HTTP server |
| `DB_PATH` | `data/vps-rental.db` | Path SQLite database |
| `ADMIN_USER` | `admin` | Default admin username |
| `ADMIN_PASS` | `admin123` | Default admin password |

### Akses

Setelah server jalan:
- **Landing page:** `http://localhost:8201/`
- **Admin panel:** `http://localhost:8201/admin`
- **Default login:** `admin` / `admin123`

---

## 📦 Data Model

### Package (Paket)
```go
type Package struct {
    ID          string  `json:"id"`
    CategoryID  string  `json:"categoryId"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    ProductType string  `json:"productType"`  // "vps_linux", "rdp_windows", "radio"
    Price       float64 `json:"price"`        // Base price per hour (IDR)
    Cores       int     `json:"cores"`
    RamMb       int     `json:"ramMb"`
    DiskGb      int     `json:"diskGb"`
    OsType      string  `json:"osType"`
    Active      bool    `json:"active"`
    Tiers       []Tier  `json:"tiers"`       // Duration-based pricing
}
```

### Tier (Durasi Harga)
```go
type Tier struct {
    DurationHours int     `json:"durationHours"`
    Price         float64 `json:"price"`     // Total price for this duration
    DiscountPct   float64 `json:"discountPct"`
    Label         string  `json:"label"`     // "1 Jam", "6 Jam", "1 Hari", dll
}
```

### Order (Pesanan)
```go
type Order struct {
    ID            string  `json:"id"`
    PackageID     string  `json:"packageId"`
    Email         string  `json:"email"`
    WhatsApp      string  `json:"whatsapp"`
    DurationHours int     `json:"durationHours"`
    BasePrice     float64 `json:"basePrice"`
    Discount       float64 `json:"discount"`
    TotalPrice    float64 `json:"totalPrice"`
    Status        string  `json:"status"`    // pending, paid, active, expired
    CouponCode    string  `json:"couponCode"`
    ProxmoxVMID   int     `json:"proxmoxVmid"`
    CreatedAt     string  `json:"createdAt"`
    ExpiresAt     string  `json:"expiresAt"`
}
```

---

## 🔧 Deploy dengan systemd

### 1. Copy binary
```bash
mkdir -p /opt/vps-rental/data
cp vps-rental /opt/vps-rental/
chmod +x /opt/vps-rental/vps-rental
```

### 2. Buat systemd service
```bash
sudo tee /etc/systemd/system/vps-rental.service << 'EOF'
[Unit]
Description=VPS Rental Web App
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/vps-rental
Environment=PORT=8201
Environment=DB_PATH=/opt/vps-rental/data/vps-rental.db
ExecStart=/opt/vps-rental/vps-rental
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

### 3. Enable & start
```bash
sudo systemctl daemon-reload
sudo systemctl enable vps-rental
sudo systemctl start vps-rental
```

### 4. Verifikasi
```bash
systemctl status vps-rental
curl http://localhost:8201/health
```

---

## 🔗 Konfigurasi Proxmox

1. Buka **Admin Panel** → **Settings**
2. Isi konfigurasi Proxmox:
   - **Host:** IP Proxmox VE (contoh: `192.168.1.100`)
   - **Port:** `8006` (default Proxmox)
   - **User:** `root@pam` (atau user PVE lain)
   - **Password:** password Proxmox
   - **Node:** nama node Proxmox (contoh: `pve`)
   - **Storage:** storage untuk LXC (contoh: `local-lvm`)
3. Klik **Save**
4. Buka tab **Proxmox** → cek status koneksi
5. List VM/LXC akan muncul, bisa Start/Stop/Delete/Create

### Create LXC Container dari Admin
1. Pilih paket VPS
2. Klik **Create** → sistem auto-generate VMID
3. Container dibuat dengan specs sesuai paket (cores, RAM, disk)
4. Password root auto-generated, ditampilkan di panel
5. Customer bisa SSH/RDP ke container

---

## 🧪 Testing

```bash
# Build check
go build -o /tmp/vps-rental .

# Start server
PORT=18201 DB_PATH=/tmp/test.db /tmp/vps-rental &

# Smoke tests
curl http://localhost:18201/health
curl http://localhost:18201/api/v1/packages | jq .
curl -X POST http://localhost:18201/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","whatsapp":"+62","packageId":"pkg-1","durationHours":1}'
```

---

## 🔐 Keamanan

- Password admin di-hash dengan **bcrypt** (cost 10)
- Session token: **32-byte random hex**, expiry 24 jam
- Admin API dilindungi middleware `adminAuth`
- SQL injection: menggunakan `database/sql` parameterized queries
- CORS: default same-origin (admin & public di domain yang sama)
- Proxmox credentials disimpan di DB (untuk production, gunakan encryption at rest)

---

## 📝 Demo Data

Saat pertama kali dijalankan, sistem auto-seed:
- **4 kategori:** VPS Linux, RDP Windows, Radio Streaming, Add-ons
- **9 paket:** Cupang, Gurame, Arwana, Jangkrik, Kenari, Murai, Cendrawasih, dll
- **8 tiers per paket:** 1 jam, 6 jam, 12 jam, 1 hari, 3 hari, 7 hari, 14 hari, 30 hari
- **1 kupon demo:** `HEMAT10` (diskon 10%)
- **1 admin:** `admin` / `admin123`

---

## 🤝 Kontribusi

1. Fork repository
2. Buat branch fitur (`git checkout -b feature/nama-fitur`)
3. Commit perubahan (`git commit -m 'Tambah fitur X'`)
4. Push branch (`git push origin feature/nama-fitur`)
5. Buat Pull Request

---

## 📄 Lisensi

MIT License — bebas digunakan, dimodifikasi, dan distribusikan.

---

## 👨‍💻 Author

**Abdul Hasan**

Dibuat dengan ❤️ menggunakan Go + Tailwind CSS
