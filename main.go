package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"vps-rental/api"
	"vps-rental/models"
	"vps-rental/store"
)

//go:embed static
var staticFS embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8097"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/vps-rental.db"
	}

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()

	h := api.New(s)

	// Seed demo data
	seedDemoData(s)

	mux := http.NewServeMux()

	// Register all API routes
	h.RegisterRoutes(mux)

	// Static files (CSS, JS, images)
	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	// Landing page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(landingPage))
	})

	// Admin panel
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(adminPage))
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "OK")
	})

	log.Printf("🚀 VPS Rental starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func seedDemoData(s *store.Store) {
	// Check if packages already exist
	pkgs, _ := s.GetPackages()
	if len(pkgs) > 0 {
		return
	}
	log.Println("📦 Seeding demo data...")

	// Categories
	type catSeed struct{ id, name string; sort int }
	cats := []catSeed{
		{"cat-vps-linux", "VPS Linux Paket Hemat", 0},
		{"cat-rdp-windows", "RDP Windows Paket Hemat", 1},
		{"cat-radio", "Radio Streaming Icecast", 2},
		{"cat-promo", "Promo", 3},
	}
	for _, c := range cats {
		s.SaveCategory(&models.Category{ID: c.id, Name: c.name, SortOrder: c.sort})
	}

	// Helper to make tiers
	makeTiers := func(baseHourly float64) []models.Tier {
		return []models.Tier{
			{1, "1 Jam", baseHourly, baseHourly, 0, "Instan"},
			{3, "3 Jam", baseHourly * 3 * 0.62, baseHourly * 0.62, 38, ""},
			{6, "6 Jam", baseHourly * 6 * 0.44, baseHourly * 0.44, 56, ""},
			{12, "12 Jam", baseHourly * 12 * 0.33, baseHourly * 0.33, 67, ""},
			{24, "1 Hari", baseHourly * 24 * 0.22, baseHourly * 0.22, 78, "Populer"},
			{72, "3 Hari", baseHourly * 72 * 0.17, baseHourly * 0.17, 83, ""},
			{168, "7 Hari", baseHourly * 168 * 0.13, baseHourly * 0.13, 87, "Hemat"},
			{720, "1 Bulan", baseHourly * 720 * 0.093, baseHourly * 0.093, 91, "Best Value"},
		}
	}

	type pkgSeed struct {
		name, desc, catID, productType, osType, vmType string
		price, cores, ram, disk                       int
		hourlyBase                                     float64
	}
	packages := []pkgSeed{
		{"Paket Teri Medan", "CPU 1 vCore, RAM 0.5 Giga, Storage 10 GB, OS Linux", "cat-vps-linux", "PROXMOX_VPS", "Debian 13", "lxc", 15000, 1, 512, 10, 225},
		{"Paket Lele", "CPU 1 vCore, RAM 1 Giga, Storage 10 GB, OS Linux", "cat-vps-linux", "PROXMOX_VPS", "Debian 13", "lxc", 20000, 1, 1024, 10, 300},
		{"Paket Nila", "CPU 2 vCore, RAM 2 Giga, Storage 20 GB, OS Linux", "cat-vps-linux", "PROXMOX_VPS", "Debian 13", "lxc", 40000, 2, 2048, 20, 600},
		{"Paket Patin", "CPU 2 vCore, RAM 4 Giga, Storage 40 GB, OS Linux", "cat-vps-linux", "PROXMOX_VPS", "Ubuntu 24.04", "lxc", 60000, 2, 4096, 40, 900},
		{"Paket Cupang", "CPU 2 vCore, RAM 2 Giga, Storage 20 GB, Windows Server 2012", "cat-rdp-windows", "PROXMOX_VPS", "Windows Server 2012", "kvm", 500, 2, 2048, 20, 500},
		{"Paket Gurame", "CPU 2 vCore, RAM 2 Giga, Storage 20 GB, OS Linux", "cat-rdp-windows", "PROXMOX_VPS", "Debian 13", "lxc", 750, 2, 2048, 20, 750},
		{"Paket Arwana", "CPU 4 vCore, RAM 4 Giga, Storage 40 GB, Windows Server 2012", "cat-rdp-windows", "PROXMOX_VPS", "Windows Server 2012", "kvm", 1000, 4, 4096, 40, 1000},
		{"Paket Jangkrik", "Radio Streaming 64 kbps, CPU 1 vCore, RAM 2 GB", "cat-radio", "RADIO_STREAMING", "Ubuntu 24.04", "lxc", 1500, 1, 2048, 20, 1500},
		{"Paket Kenari", "Radio Streaming 128 kbps, CPU 1 vCore, RAM 2 GB", "cat-radio", "RADIO_STREAMING", "Ubuntu 24.04", "lxc", 3000, 1, 2048, 20, 3000},
	}

	for i, p := range packages {
		pkg := &models.Package{
			ID:              fmt.Sprintf("pkg-%d", i+1),
			Name:            p.name,
			Description:     p.desc,
			CategoryID:      p.catID,
			ProductType:     p.productType,
			Price:           p.price,
			MonthlyAnchor:   p.price,
			PricingStrategy: "COMPETITIVE",
			Cores:           p.cores,
			RamMB:           p.ram,
			DiskGB:          p.disk,
			OSType:          p.osType,
			VMType:          p.vmType,
			IsAvailable:     true,
			Tiers:           makeTiers(p.hourlyBase),
		}
		s.SavePackage(pkg)
	}

	// Demo coupons
	s.SaveCoupon(&models.Coupon{ID: "coupon1", Code: "HEMAT10", DiscountPct: 10, MaxUses: 100, IsActive: true})

	// Default settings
	s.SetSetting("brand_name", "ECER VPS/RDP")
	s.SetSetting("brand_tagline", "Sewa VPS & Remote Desktop Instan")
	s.SetSetting("whatsapp", "+628****3087")
	s.SetSetting("email", "admin@ecer.biz.id")

	log.Println("✅ Demo data seeded")
}

// Landing page HTML (inline for go:embed simplicity)
var landingPage = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>ECER VPS/RDP — Sewa Remote Desktop Instan</title>
<script src="https://cdn.tailwindcss.com"></script>
<link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">
<script>
tailwind.config={theme:{extend:{fontFamily:{sans:['Outfit','sans-serif']},colors:{emerald:{950:'#002c22'}}}}}
</script>
<style>body{font-family:'Outfit',sans-serif}</style>
</head>
<body class="bg-white text-gray-900">

<!-- NAVBAR -->
<nav class="sticky top-0 z-50 bg-white/95 backdrop-blur border-b border-gray-100">
<div class="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
<div class="flex items-center gap-2">
<div class="w-8 h-8 rounded-lg bg-emerald-500 flex items-center justify-center text-white font-bold text-sm">E</div>
<span class="font-bold text-lg">ECER <span class="text-emerald-600">VPS/RDP</span></span>
</div>
<div class="hidden md:flex items-center gap-6 text-sm font-medium text-gray-600">
<a href="#" class="hover:text-emerald-600">🏠 Beranda</a>
<a href="#features" class="hover:text-emerald-600">📦 Produk</a>
<a href="#" class="hover:text-emerald-600">🤝 Kemitraan</a>
<a href="/admin" class="hover:text-emerald-600">👤 Akses Portal</a>
<a href="#" class="hover:text-emerald-600">🔍 Lacak</a>
</div>
<button onclick="document.getElementById('mobile-menu').classList.toggle('hidden')" class="md:hidden text-2xl">☰</button>
</div>
<div id="mobile-menu" class="hidden md:hidden px-4 pb-4 space-y-2 text-sm">
<a href="#" class="block py-2">🏠 Beranda</a>
<a href="#features" class="block py-2">📦 Produk</a>
<a href="/admin" class="block py-2">👤 Akses Portal</a>
</div>
</nav>

<!-- HERO -->
<section class="bg-gradient-to-br from-emerald-950 via-emerald-900 to-emerald-800 text-white py-20 px-4">
<div class="max-w-4xl mx-auto text-center">
<span class="inline-block bg-emerald-500/20 text-emerald-300 text-xs font-semibold px-3 py-1 rounded-full mb-4">⚡ Aktivasi Otomatis 24/7</span>
<h1 class="text-4xl md:text-5xl font-extrabold mb-4 leading-tight">Layanan RDP & VPS Otomatis<br>Berkualitas Tinggi</h1>
<p class="text-emerald-200 text-lg mb-2 font-medium">Komputasi RDP Terisolasi — Performa Cepat Tanpa Batas</p>
<p class="text-emerald-300/80 max-w-2xl mx-auto mb-8">Dirancang untuk kebutuhan komputasi harian, streaming, dan pemantauan bot secara non-stop. Sistem kami otomatis menyiapkan akun RDP & VPS Anda secara <strong class="text-white">instan</strong> begitu transaksi selesai.</p>
<div class="flex flex-wrap gap-3 justify-center">
<a href="#order" class="bg-emerald-500 hover:bg-emerald-600 text-white font-semibold px-6 py-3 rounded-xl transition">🚀 Mulai Sewa Sekarang</a>
<a href="#" class="bg-white/10 hover:bg-white/20 text-white font-semibold px-6 py-3 rounded-xl transition">💬 Chat WhatsApp</a>
</div>
</div>
</section>

<!-- FEATURES -->
<section id="features" class="py-16 px-4 bg-gray-50">
<div class="max-w-6xl mx-auto">
<h2 class="text-center text-2xl font-bold mb-10">Layanan RDP & VPS Otomatis Berkualitas Tinggi</h2>
<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
<div class="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm hover:shadow-md hover:border-emerald-200 transition">
<div class="text-3xl mb-3">🖥️</div>
<h3 class="font-bold mb-2">Akses Desktop Windows</h3>
<p class="text-gray-500 text-sm">Lingkungan Windows Server yang stabil untuk semua aplikasi komputasi Anda.</p>
</div>
<div class="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm hover:shadow-md hover:border-emerald-200 transition">
<div class="text-3xl mb-3">🔐</div>
<h3 class="font-bold mb-2">Keamanan Terlindungi</h3>
<p class="text-gray-500 text-sm">Setiap pelanggan memiliki akun pengguna atau Container VPS terisolasi mandiri.</p>
</div>
<div class="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm hover:shadow-md hover:border-emerald-200 transition">
<div class="text-3xl mb-3">🌍</div>
<h3 class="font-bold mb-2">Koneksi Server Stabil</h3>
<p class="text-gray-500 text-sm">Terhubung langsung ke jaringan internet berkecepatan tinggi dengan latensi rendah.</p>
</div>
<div class="bg-white rounded-2xl border border-gray-100 p-6 shadow-sm hover:shadow-md hover:border-emerald-200 transition">
<div class="text-3xl mb-3">⚡</div>
<h3 class="font-bold mb-2">Aktivasi Instan Otomatis</h3>
<p class="text-gray-500 text-sm">RDP Anda otomatis aktif dan siap digunakan dalam beberapa detik setelah bayar.</p>
</div>
</div>
</div>
</section>

<!-- HOW IT WORKS -->
<section class="py-16 px-4">
<div class="max-w-4xl mx-auto text-center">
<h2 class="text-2xl font-bold mb-10">Alur Pemesanan Cepat</h2>
<div class="grid grid-cols-1 md:grid-cols-3 gap-8">
<div>
<div class="w-12 h-12 rounded-full bg-emerald-100 text-emerald-700 font-bold text-xl flex items-center justify-center mx-auto mb-3">1</div>
<h3 class="font-bold mb-2">Pilih Paket</h3>
<p class="text-gray-500 text-sm">Tentukan spesifikasi & durasi yang sesuai kebutuhan.</p>
</div>
<div>
<div class="w-12 h-12 rounded-full bg-emerald-100 text-emerald-700 font-bold text-xl flex items-center justify-center mx-auto mb-3">2</div>
<h3 class="font-bold mb-2">Bayar Otomatis</h3>
<p class="text-gray-500 text-sm">Scan QRIS instan atau transfer via gateway pembayaran.</p>
</div>
<div>
<div class="w-12 h-12 rounded-full bg-emerald-100 text-emerald-700 font-bold text-xl flex items-center justify-center mx-auto mb-3">3</div>
<h3 class="font-bold mb-2">Langsung Aktif</h3>
<p class="text-gray-500 text-sm">Detail login RDP langsung dikirim ke WhatsApp & email.</p>
</div>
</div>
</div>
</section>

<!-- ORDER FORM -->
<section id="order" class="py-16 px-4 bg-gray-50">
<div class="max-w-xl mx-auto">
<div class="bg-white rounded-2xl border border-gray-200 p-6 md:p-8 shadow-sm">
<h2 class="text-xl font-bold mb-1">Formulir Sewa RDP & VPS</h2>
<p class="text-gray-500 text-sm mb-6">Isi data di bawah untuk memulai penyediaan server instan.</p>

<div class="space-y-4">
<div>
<label class="text-sm font-medium text-gray-700 block mb-1">Email</label>
<input id="f-email" type="email" class="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none" placeholder="email@contoh.com">
</div>
<div>
<label class="text-sm font-medium text-gray-700 block mb-1">WhatsApp</label>
<input id="f-wa" type="tel" class="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none" placeholder="+62812xxxx">
</div>
<div>
<label class="text-sm font-medium text-gray-700 block mb-1">Pilih Paket</label>
<select id="f-pkg" class="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none bg-white" onchange="onPackageChange()">
<option value="">Memuat daftar paket...</option>
</select>
</div>

<!-- Duration Tiers -->
<div id="tiers-section" class="hidden">
<label class="text-sm font-medium text-gray-700 block mb-2">Durasi Sewa</label>
<div id="tiers-grid" class="grid grid-cols-2 sm:grid-cols-3 gap-2"></div>
</div>

<div>
<label class="text-sm font-medium text-gray-700 block mb-1">Kode Kupon <span class="text-gray-400">(opsional)</span></label>
<div class="flex gap-2">
<input id="f-coupon" type="text" class="flex-1 border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 outline-none" placeholder="KODEKUPON">
<button onclick="applyCoupon()" class="bg-gray-100 hover:bg-gray-200 text-gray-700 font-medium px-4 py-2.5 rounded-xl text-sm transition">Gunakan</button>
</div>
</div>

<div class="bg-emerald-50 rounded-xl p-4">
<div class="text-sm text-gray-600">Total Investasi</div>
<div id="total-display" class="text-2xl font-bold text-emerald-700">Rp 0</div>
<div id="total-detail" class="text-xs text-gray-500 mt-1"></div>
</div>

<button id="btn-order" onclick="submitOrder()" class="w-full bg-emerald-500 hover:bg-emerald-600 text-white font-semibold py-3 rounded-xl transition text-sm disabled:opacity-50 disabled:cursor-not-allowed" disabled>
Lanjutkan Pembayaran →
</button>
</div>
</div>
</div>
</section>

<!-- FOOTER -->
<footer class="bg-emerald-950 text-white py-12 px-4">
<div class="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-8">
<div>
<div class="flex items-center gap-2 mb-3">
<div class="w-8 h-8 rounded-lg bg-emerald-500 flex items-center justify-center text-white font-bold text-sm">E</div>
<span class="font-bold text-lg">ECER VPS/RDP</span>
</div>
<p class="text-emerald-300/80 text-sm">Penyedia sewa VPS dan Remote Desktop Protocol (RDP) otomatis, instan, dan terjangkau untuk kebutuhan komputasi Anda.</p>
<p class="text-emerald-400/50 text-xs mt-4">© 2026 ECER VPS/RDP. All rights reserved.</p>
</div>
<div>
<h4 class="font-bold mb-3">Hubungi Kami / Support</h4>
<p class="text-emerald-300/80 text-sm mb-1">📧 admin@ecer.biz.id</p>
<p class="text-emerald-300/80 text-sm">📞 WhatsApp: +62 813-8118-3087</p>
</div>
</div>
</footer>

<script>
let packages=[],selectedPkg=null,selectedTier=null,couponDiscount=0;

async function loadPackages(){
  try{
    const r=await fetch('/api/v1/packages');
    packages=await r.json();
    const sel=document.getElementById('f-pkg');
    sel.innerHTML='<option value="">— Pilih Paket —</option>';
    const cats={};
    packages.forEach(p=>{
      if(!cats[p.categoryId])cats[p.categoryId]=[];
      cats[p.categoryId].push(p);
    });
    for(const[cat,ps]of Object.entries(cats)){
      const og=document.createElement('optgroup');
      og.label=ps[0]?.productType||cat;
      ps.forEach(p=>{
        const o=document.createElement('option');
        o.value=p.id;
        o.textContent=p.name+' — '+p.description;
        og.appendChild(o);
      });
      sel.appendChild(og);
    }
  }catch(e){console.error(e)}
}

function onPackageChange(){
  const id=document.getElementById('f-pkg').value;
  selectedPkg=packages.find(p=>p.id===id)||null;
  selectedTier=null;couponDiscount=0;
  const sec=document.getElementById('tiers-section');
  const grid=document.getElementById('tiers-grid');
  if(!selectedPkg||!selectedPkg.tiers||!selectedPkg.tiers.length){sec.classList.add('hidden');updateTotal();return}
  sec.classList.remove('hidden');
  grid.innerHTML='';
  selectedPkg.tiers.forEach(t=>{
    const btn=document.createElement('button');
    btn.className='border border-gray-200 rounded-xl p-3 text-left hover:border-emerald-300 transition text-sm';
    const badge=t.badge?'<span class="inline-block bg-emerald-100 text-emerald-700 text-[10px] font-bold px-2 py-0.5 rounded-full mb-1">'+t.badge+'</span>':'';
    btn.innerHTML=badge+'<div class="font-bold">'+t.label+'</div><div class="text-emerald-600 font-semibold">Rp '+Math.round(t.price).toLocaleString('id')+'</div>'+(t.discountPct>0?'<div class="text-gray-400 text-xs">-'+t.discountPct+'%</div>':'');
    btn.onclick=()=>{selectedTier=t;document.querySelectorAll('#tiers-grid button').forEach(b=>b.classList.remove('border-emerald-500','bg-emerald-50'));btn.classList.add('border-emerald-500','bg-emerald-50');updateTotal()};
    grid.appendChild(btn);
  });
}

async function applyCoupon(){
  const code=document.getElementById('f-coupon').value.trim();
  if(!code||!selectedTier)return;
  try{
    const r=await fetch('/api/v1/coupons/validate',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({code,price:selectedTier.price})});
    const d=await r.json();
    if(d.valid){couponDiscount=d.discount;updateTotal()}else{alert(d.error||'Kupon tidak valid')}
  }catch(e){alert('Gagal validasi kupon')}
}

function updateTotal(){
  const btn=document.getElementById('btn-order');
  const disp=document.getElementById('total-display');
  const det=document.getElementById('total-detail');
  if(!selectedTier){disp.textContent='Rp 0';det.textContent='';btn.disabled=true;return}
  let total=selectedTier.price-couponDiscount;
  if(total<0)total=0;
  disp.textContent='Rp '+Math.round(total).toLocaleString('id');
  let detail=selectedTier.hourlyRate.toFixed(0)+'/jam × '+selectedTier.label;
  if(couponDiscount>0)detail+=' (kupon -Rp'+Math.round(couponDiscount).toLocaleString('id')+')';
  det.textContent=detail;
  btn.disabled=false;
}

async function submitOrder(){
  const email=document.getElementById('f-email').value;
  const wa=document.getElementById('f-wa').value;
  if(!email||!wa){alert('Email dan WhatsApp wajib diisi');return}
  if(!selectedPkg||!selectedTier){alert('Pilih paket dan durasi');return}
  const btn=document.getElementById('btn-order');
  btn.disabled=true;btn.textContent='Memproses...';
  try{
    const r=await fetch('/api/v1/orders',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email,whatsapp:wa,packageId:selectedPkg.id,durationHours:selectedTier.durationHours,couponCode:document.getElementById('f-coupon').value.trim()})});
    const d=await r.json();
    if(d.error){alert(d.error)}else{alert('Pesanan dibuat! ID: '+d.id+'\nTotal: Rp '+Math.round(d.totalPrice).toLocaleString('id')+'\n\nStatus: '+d.status)}
  }catch(e){alert('Gagal membuat pesanan: '+e.message)}
  btn.disabled=false;btn.textContent='Lanjutkan Pembayaran →';
}

loadPackages();
</script>
</body>
</html>`

// Admin panel HTML
var adminPage = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Admin — VPS Rental</title>
<script src="https://cdn.tailwindcss.com"></script>
<link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap" rel="stylesheet">
<script>tailwind.config={theme:{extend:{fontFamily:{sans:['Outfit','sans-serif']},colors:{emerald:{950:'#002c22'}}}}}</script>
<style>body{font-family:'Outfit',sans-serif}</style>
</head>
<body class="bg-gray-50 min-h-screen">

<!-- LOGIN SCREEN -->
<div id="login-screen" class="min-h-screen flex items-center justify-center">
<div class="bg-white rounded-2xl border border-gray-200 p-8 w-full max-w-sm shadow-sm">
<div class="text-center mb-6">
<div class="w-12 h-12 rounded-xl bg-emerald-500 flex items-center justify-center text-white font-bold text-lg mx-auto mb-3">E</div>
<h1 class="text-xl font-bold">Admin Panel</h1>
<p class="text-gray-500 text-sm">Masuk untuk mengelola layanan</p>
</div>
<div class="space-y-3">
<input id="l-user" type="text" placeholder="Username" class="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm">
<input id="l-pass" type="password" placeholder="Password" class="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm">
<button onclick="doLogin()" class="w-full bg-emerald-500 hover:bg-emerald-600 text-white font-semibold py-2.5 rounded-xl text-sm transition">Masuk</button>
</div>
<p id="login-err" class="text-red-500 text-xs mt-2 hidden"></p>
</div>
</div>

<!-- ADMIN DASHBOARD -->
<div id="admin-app" class="hidden">
<!-- Top bar -->
<div class="bg-white border-b border-gray-200 px-4 h-14 flex items-center justify-between">
<span class="font-bold text-emerald-700">🛡️ VPS Rental Admin</span>
<button onclick="doLogout()" class="text-gray-500 hover:text-red-500 text-sm">Logout</button>
</div>

<div class="flex">
<!-- Sidebar -->
<aside class="w-56 bg-white border-r border-gray-200 min-h-[calc(100vh-56px)] p-4 space-y-1 hidden md:block">
<button onclick="showTab('dashboard')" class="sidebar-btn w-full text-left px-3 py-2 rounded-lg text-sm font-medium hover:bg-emerald-50 hover:text-emerald-700">📊 Dashboard</button>
<button onclick="showTab('packages')" class="sidebar-btn w-full text-left px-3 py-2 rounded-lg text-sm font-medium hover:bg-emerald-50 hover:text-emerald-700">📦 Paket</button>
<button onclick="showTab('orders')" class="sidebar-btn w-full text-left px-3 py-2 rounded-lg text-sm font-medium hover:bg-emerald-50 hover:text-emerald-700">🧾 Pesanan</button>
<button onclick="showTab('coupons')" class="sidebar-btn w-full text-left px-3 py-2 rounded-lg text-sm font-medium hover:bg-emerald-50 hover:text-emerald-700">🎫 Kupon</button>
<button onclick="showTab('proxmox')" class="sidebar-btn w-full text-left px-3 py-2 rounded-lg text-sm font-medium hover:bg-emerald-50 hover:text-emerald-700">🖥️ Proxmox</button>
<button onclick="showTab('settings')" class="sidebar-btn w-full text-left px-3 py-2 rounded-lg text-sm font-medium hover:bg-emerald-50 hover:text-emerald-700">⚙️ Settings</button>
</aside>

<!-- Mobile tabs -->
<div class="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 flex justify-around py-2 z-50">
<button onclick="showTab('dashboard')" class="text-xs text-gray-600">📊</button>
<button onclick="showTab('packages')" class="text-xs text-gray-600">📦</button>
<button onclick="showTab('orders')" class="text-xs text-gray-600">🧾</button>
<button onclick="showTab('coupons')" class="text-xs text-gray-600">🎫</button>
<button onclick="showTab('proxmox')" class="text-xs text-gray-600">🖥️</button>
<button onclick="showTab('settings')" class="text-xs text-gray-600">⚙️</button>
</div>

<!-- Main content -->
<main class="flex-1 p-6 pb-20 md:pb-6">
<!-- Dashboard -->
<div id="tab-dashboard" class="tab-content">
<h2 class="text-xl font-bold mb-4">Dashboard</h2>
<div id="stats-grid" class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6"></div>
</div>

<!-- Packages -->
<div id="tab-packages" class="tab-content hidden">
<div class="flex items-center justify-between mb-4">
<h2 class="text-xl font-bold">Paket</h2>
<button onclick="openPkgForm()" class="bg-emerald-500 hover:bg-emerald-600 text-white text-sm font-medium px-4 py-2 rounded-lg">+ Tambah Paket</button>
</div>
<div id="pkg-table" class="bg-white rounded-xl border border-gray-200 overflow-hidden">
<table class="w-full text-sm"><thead class="bg-gray-50"><tr><th class="px-4 py-3 text-left">Nama</th><th class="px-4 py-3">Kategori</th><th class="px-4 py-3">Spek</th><th class="px-4 py-3">Harga</th><th class="px-4 py-3">Aksi</th></tr></thead><tbody id="pkg-tbody"></tbody></table>
</div>
</div>

<!-- Orders -->
<div id="tab-orders" class="tab-content hidden">
<h2 class="text-xl font-bold mb-4">Pesanan</h2>
<div class="bg-white rounded-xl border border-gray-200 overflow-hidden">
<table class="w-full text-sm"><thead class="bg-gray-50"><tr><th class="px-4 py-3 text-left">ID</th><th class="px-4 py-3">Paket</th><th class="px-4 py-3">Total</th><th class="px-4 py-3">Status</th><th class="px-4 py-3">Aksi</th></tr></thead><tbody id="order-tbody"></tbody></table>
</div>
</div>

<!-- Coupons -->
<div id="tab-coupons" class="tab-content hidden">
<div class="flex items-center justify-between mb-4">
<h2 class="text-xl font-bold">Kupon</h2>
<button onclick="openCouponForm()" class="bg-emerald-500 hover:bg-emerald-600 text-white text-sm font-medium px-4 py-2 rounded-lg">+ Tambah Kupon</button>
</div>
<div id="coupon-list" class="grid grid-cols-1 md:grid-cols-2 gap-4"></div>
</div>

<!-- Proxmox -->
<div id="tab-proxmox" class="tab-content hidden">
<h2 class="text-xl font-bold mb-4">Proxmox VE</h2>
<div id="px-status" class="bg-white rounded-xl border border-gray-200 p-4 mb-4"></div>
<div id="px-vms" class="bg-white rounded-xl border border-gray-200 overflow-hidden">
<table class="w-full text-sm"><thead class="bg-gray-50"><tr><th class="px-4 py-3">VMID</th><th class="px-4 py-3">Nama</th><th class="px-4 py-3">Status</th><th class="px-4 py-3">Spek</th><th class="px-4 py-3">Aksi</th></tr></thead><tbody id="px-tbody"></tbody></table>
</div>
</div>

<!-- Settings -->
<div id="tab-settings" class="tab-content hidden">
<h2 class="text-xl font-bold mb-4">Settings</h2>
<div class="bg-white rounded-xl border border-gray-200 p-6 max-w-lg space-y-4">
<div><label class="text-sm font-medium block mb-1">Brand Name</label><input id="s-brand" class="w-full border rounded-lg px-3 py-2 text-sm"></div>
<div><label class="text-sm font-medium block mb-1">WhatsApp</label><input id="s-wa" class="w-full border rounded-lg px-3 py-2 text-sm"></div>
<div><label class="text-sm font-medium block mb-1">Email</label><input id="s-email" class="w-full border rounded-lg px-3 py-2 text-sm"></div>
<hr>
<h3 class="font-bold">Proxmox Connection</h3>
<div><label class="text-sm font-medium block mb-1">Host</label><input id="s-px-host" class="w-full border rounded-lg px-3 py-2 text-sm" placeholder="192.168.1.100"></div>
<div><label class="text-sm font-medium block mb-1">Port</label><input id="s-px-port" class="w-full border rounded-lg px-3 py-2 text-sm" value="8006"></div>
<div><label class="text-sm font-medium block mb-1">User</label><input id="s-px-user" class="w-full border rounded-lg px-3 py-2 text-sm" value="root"></div>
<div><label class="text-sm font-medium block mb-1">Password</label><input id="s-px-pass" type="password" class="w-full border rounded-lg px-3 py-2 text-sm"></div>
<div><label class="text-sm font-medium block mb-1">Realm</label><select id="s-px-realm" class="w-full border rounded-lg px-3 py-2 text-sm"><option value="pam">PAM (Linux)</option><option value="pve">PVE</option></select></div>
<div><label class="text-sm font-medium block mb-1">Node</label><input id="s-px-node" class="w-full border rounded-lg px-3 py-2 text-sm" placeholder="pve"></div>
<button onclick="saveSettings()" class="bg-emerald-500 hover:bg-emerald-600 text-white font-medium px-6 py-2 rounded-lg text-sm">Simpan</button>
</div>
</div>
</main>
</div>
</div>

<script>
let authed=false;
const API='/admin/api';

// Auth
async function doLogin(){
  const u=document.getElementById('l-user').value,p=document.getElementById('l-pass').value;
  try{
    const r=await fetch(API+'/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:u,password:p})});
    const d=await r.json();
    if(d.token){authed=true;document.getElementById('login-screen').classList.add('hidden');document.getElementById('admin-app').classList.remove('hidden');loadAll()}
    else{document.getElementById('login-err').textContent=d.error||'Login gagal';document.getElementById('login-err').classList.remove('hidden')}
  }catch(e){alert('Error: '+e.message)}
}
async function doLogout(){
  await fetch(API+'/logout');
  authed=false;location.reload();
}

// Tabs
function showTab(t){
  document.querySelectorAll('.tab-content').forEach(el=>el.classList.add('hidden'));
  document.getElementById('tab-'+t).classList.remove('hidden');
  document.querySelectorAll('.sidebar-btn').forEach(b=>b.classList.remove('bg-emerald-50','text-emerald-700'));
  event.target.closest('button')?.classList.add('bg-emerald-50','text-emerald-700');
  if(t==='dashboard')loadStats();
  if(t==='packages')loadPackages();
  if(t==='orders')loadOrders();
  if(t==='coupons')loadCoupons();
  if(t==='proxmox')loadProxmox();
  if(t==='settings')loadSettings();
}

// Stats
async function loadStats(){
  const r=await fetch(API+'/stats');const d=await r.json();
  const g=document.getElementById('stats-grid');
  const labels={total_orders:'Total Pesanan',active_orders:'Aktif',pending_orders:'Pending',active_packages:'Paket Aktif',active_coupons:'Kupon Aktif',total_revenue:'Pendapatan'};
  g.innerHTML='';
  for(const[k,v]of Object.entries(d)){
    const val=k==='total_revenue'?'Rp '+Number(v).toLocaleString('id'):v;
    g.innerHTML+='<div class="bg-white rounded-xl border border-gray-200 p-4"><div class="text-gray-500 text-xs">'+(labels[k]||k)+'</div><div class="text-2xl font-bold text-emerald-700">'+val+'</div></div>';
  }
}

// Packages
async function loadPackages(){
  const r=await fetch(API+'/packages');const pkgs=await r.json();
  const tb=document.getElementById('pkg-tbody');
  tb.innerHTML='';
  pkgs.forEach(p=>{
    const status=p.isAvailable?'<span class="text-emerald-600">✓</span>':'<span class="text-red-500">✗</span>';
    tb.innerHTML+='<tr class="border-t border-gray-100"><td class="px-4 py-3 font-medium">'+p.name+'</td><td class="px-4 py-3 text-center text-xs">'+p.productType+'</td><td class="px-4 py-3 text-center text-xs">'+p.cores+'c/'+p.ramMb+'M/'+p.diskGb+'G</td><td class="px-4 py-3 text-center">Rp '+p.price.toLocaleString('id')+'</td><td class="px-4 py-3 text-center"><button onclick="deletePkg(\''+p.id+'\')" class="text-red-500 text-xs hover:underline">Hapus</button></td></tr>';
  });
}
async function deletePkg(id){if(!confirm('Hapus paket ini?'))return;await fetch(API+'/packages/'+id,{method:'DELETE'});loadPackages()}
function openPkgForm(){
  const name=prompt('Nama Paket (mis: Paket Lele):');if(!name)return;
  const desc=prompt('Deskripsi (mis: CPU 1 vCore, RAM 1 GB):')||'';
  const cat=prompt('Kategori ID (cat-vps-linux / cat-rdp-windows / cat-radio):')||'cat-vps-linux';
  const type=prompt('Tipe (PROXMOX_VPS / RDP_WINDOWS / RADIO_STREAMING):')||'PROXMOX_VPS';
  const cores=parseInt(prompt('Cores:',1))||1;
  const ram=parseInt(prompt('RAM (MB):',1024))||1024;
  const disk=parseInt(prompt('Disk (GB):',10))||10;
  const price=parseInt(prompt('Harga bulanan (Rp):',20000))||20000;
  const os=prompt('OS (Debian 13 / Windows Server 2012):')||'Debian 13';
  const vm=prompt('VM Type (lxc / kvm):')||'lxc';
  fetch(API+'/packages',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name,description:desc,categoryId:cat,productType:type,price,monthlyAnchor:price,cores,ramMb:ram,diskGb:disk,osType:os,vmType:vm,isAvailable:true,stockLimit:0,currentStock:0,tiers:[]})}).then(()=>loadPackages());
}

// Orders
async function loadOrders(){
  const r=await fetch(API+'/orders');const orders=await r.json();
  const tb=document.getElementById('order-tbody');
  tb.innerHTML='';
  orders.forEach(o=>{
    const colors={pending:'bg-yellow-100 text-yellow-700',paid:'bg-blue-100 text-blue-700',active:'bg-emerald-100 text-emerald-700',expired:'bg-gray-100 text-gray-500',cancelled:'bg-red-100 text-red-700',provisioning:'bg-purple-100 text-purple-700'};
    tb.innerHTML+='<tr class="border-t border-gray-100"><td class="px-4 py-3 text-xs font-mono">'+o.id.slice(0,8)+'…</td><td class="px-4 py-3 text-xs">'+o.packageId+'</td><td class="px-4 py-3 text-center">Rp '+Math.round(o.totalPrice).toLocaleString('id')+'</td><td class="px-4 py-3 text-center"><span class="text-xs font-medium px-2 py-1 rounded-full '+(colors[o.status]||'')+'">'+o.status+'</span></td><td class="px-4 py-3 text-center"><select onchange="updateOrderStatus(\''+o.id+'\',this.value)" class="text-xs border rounded px-2 py-1"><option value="pending" '+(o.status==='pending'?'selected':'')+'>Pending</option><option value="paid" '+(o.status==='paid'?'selected':'')+'>Paid</option><option value="provisioning" '+(o.status==='provisioning'?'selected':'')+'>Provisioning</option><option value="active" '+(o.status==='active'?'selected':'')+'>Active</option><option value="expired" '+(o.status==='expired'?'selected':'')+'>Expired</option><option value="cancelled" '+(o.status==='cancelled'?'selected':'')+'>Cancelled</option></select></td></tr>';
  });
}
async function updateOrderStatus(id,status){await fetch(API+'/orders',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({id,status})});loadOrders()}

// Coupons
async function loadCoupons(){
  const r=await fetch(API+'/coupons');const coupons=await r.json();
  const el=document.getElementById('coupon-list');
  el.innerHTML='';
  coupons.forEach(c=>{
    const exp=c.expiresAt?'Exp: '+new Date(c.expiresAt).toLocaleDateString():'No expiry';
    const disc=c.discountPct>0?c.discountPct+'%':'Rp '+Math.round(c.discountAmt).toLocaleString('id');
    el.innerHTML+='<div class="bg-white rounded-xl border border-gray-200 p-4"><div class="flex justify-between items-start"><div><div class="font-bold text-emerald-700">'+c.code+'</div><div class="text-sm text-gray-500">Diskon: '+disc+'</div><div class="text-xs text-gray-400">'+exp+' | Terpakai: '+c.usedCount+(c.maxUses>0?'/'+c.maxUses:'∞')+'</div></div><button onclick="deleteCoupon(\''+c.id+'\')" class="text-red-500 text-xs">Hapus</button></div></div>';
  });
}
function openCouponForm(){
  const code=prompt('Kode Kupon:');if(!code)return;
  const pct=parseFloat(prompt('Diskon % (0 jika nominal):'))||0;
  const amt=parseFloat(prompt('Diskon nominal Rp (0 jika persen):'))||0;
  const max=parseInt(prompt('Max penggunaan (0=unlimited):'))||0;
  fetch(API+'/coupons',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({code,discountPct:pct,discountAmt:amt,maxUses:max,isActive:true})}).then(()=>loadCoupons());
}
async function deleteCoupon(id){if(!confirm('Hapus?'))return;await fetch(API+'/coupons/'+id,{method:'DELETE'});loadCoupons()}

// Proxmox
async function loadProxmox(){
  const r=await fetch(API+'/proxmox/status');const d=await r.json();
  document.getElementById('px-status').innerHTML=d.connected?'<span class="text-emerald-600 font-bold">✅ Terhubung ke Proxmox</span>':'<span class="text-red-500 font-bold">❌ Tidak terhubung</span> <span class="text-xs text-gray-500">— Cek Settings</span>';
  if(!d.connected){document.getElementById('px-tbody').innerHTML='<tr><td colspan="5" class="px-4 py-8 text-center text-gray-400">Proxmox tidak terhubung</td></tr>';return}
  const rv=await fetch(API+'/proxmox/vms');const vms=await rv.json();
  const tb=document.getElementById('px-tbody');
  tb.innerHTML='';
  vms.forEach(v=>{
    const statusColor=v.status==='running'?'text-emerald-600':'text-gray-400';
    tb.innerHTML+='<tr class="border-t border-gray-100"><td class="px-4 py-3 font-mono">'+v.vmid+'</td><td class="px-4 py-3">'+(v.name||'-')+'</td><td class="px-4 py-3 '+statusColor+' font-medium">'+v.status+'</td><td class="px-4 py-3 text-xs">'+v.cores+'c/'+v.ramMb+'M/'+v.diskGb+'G</td><td class="px-4 py-3 space-x-1"><button onclick="pxAction('+v.vmid+',\'start\')" class="text-emerald-600 text-xs hover:underline">▶ Start</button><button onclick="pxAction('+v.vmid+',\'stop\')" class="text-yellow-600 text-xs hover:underline">■ Stop</button><button onclick="pxAction('+v.vmid+',\'delete\')" class="text-red-500 text-xs hover:underline">✕ Del</button></td></tr>';
  });
}
async function pxAction(vmid,action){
  if(action==='delete'&&!confirm('Hapus VM '+vmid+'?'))return;
  await fetch(API+'/proxmox/action/'+vmid+'/'+action);loadProxmox();
}

// Settings
async function loadSettings(){
  const r=await fetch(API+'/settings');const d=await r.json();
  document.getElementById('s-brand').value=d.brand_name||'';
  document.getElementById('s-wa').value=d.whatsapp||'';
  document.getElementById('s-email').value=d.email||'';
  document.getElementById('s-px-host').value=d.proxmox_host||'';
  document.getElementById('s-px-port').value=d.proxmox_port||'8006';
  document.getElementById('s-px-user').value=d.proxmox_user||'root';
  document.getElementById('s-px-pass').value=d.proxmox_password||'';
  document.getElementById('s-px-realm').value=d.proxmox_realm||'pam';
  document.getElementById('s-px-node').value=d.proxmox_node||'pve';
}
async function saveSettings(){
  const data={brand_name:document.getElementById('s-brand').value,whatsapp:document.getElementById('s-wa').value,email:document.getElementById('s-email').value,proxmox_host:document.getElementById('s-px-host').value,proxmox_port:document.getElementById('s-px-port').value,proxmox_user:document.getElementById('s-px-user').value,proxmox_password:document.getElementById('s-px-pass').value,proxmox_realm:document.getElementById('s-px-realm').value,proxmox_node:document.getElementById('s-px-node').value};
  await fetch(API+'/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(data)});
  alert('Settings tersimpan!');
}

function loadAll(){loadStats();showTab('dashboard')}
</script>
</body>
</html>`
