package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"vps-rental/models"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite single writer
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.db.Close() }

func (s *Store) migrate() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS packages (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT, category_id TEXT,
			product_type TEXT DEFAULT 'PROXMOX_VPS', price INTEGER DEFAULT 0,
			monthly_anchor INTEGER DEFAULT 0, pricing_strategy TEXT DEFAULT 'COMPETITIVE',
			cores INTEGER DEFAULT 1, ram_mb INTEGER DEFAULT 512, disk_gb INTEGER DEFAULT 10,
			os_type TEXT DEFAULT 'Debian 13', vm_type TEXT DEFAULT 'lxc',
			is_available INTEGER DEFAULT 1, stock_limit INTEGER DEFAULT 0,
			current_stock INTEGER DEFAULT 0, tiers TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY, email TEXT, whatsapp TEXT, package_id TEXT,
			duration_hours INTEGER, coupon_code TEXT, original_price REAL,
			discount_amt REAL DEFAULT 0, total_price REAL, status TEXT DEFAULT 'pending',
			payment_ref TEXT, proxmox_vm_id INTEGER DEFAULT 0, proxmox_node TEXT,
			login_ip TEXT, login_user TEXT, login_pass TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			paid_at DATETIME, expires_at DATETIME)`,
		`CREATE TABLE IF NOT EXISTS coupons (
			id TEXT PRIMARY KEY, code TEXT UNIQUE, discount_pct REAL DEFAULT 0,
			discount_amt REAL DEFAULT 0, max_uses INTEGER DEFAULT 0,
			used_count INTEGER DEFAULT 0, expires_at DATETIME,
			is_active INTEGER DEFAULT 1, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS admin_users (
			id TEXT PRIMARY KEY, username TEXT UNIQUE, password TEXT, role TEXT DEFAULT 'admin')`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY, value TEXT)`,
	}
	for _, q := range schema {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// seed default admin if not exists
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	if count == 0 {
		s.db.Exec("INSERT INTO admin_users (id,username,password,role) VALUES ('admin1','admin','$2b$10$yb9AxcY0k4uqos9q4zjS/.Ef.QltZjI9h/k/fgS9i9Q70OJxc3zHm','admin')")
		// default password: "admin123"
	}
	return nil
}

// --- Categories ---

func (s *Store) GetCategories() ([]models.Category, error) {
	rows, err := s.db.Query("SELECT id, name, sort_order FROM categories ORDER BY sort_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []models.Category
	for rows.Next() {
		var c models.Category
		rows.Scan(&c.ID, &c.Name, &c.SortOrder)
		cats = append(cats, c)
	}
	return cats, nil
}

func (s *Store) SaveCategory(c *models.Category) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO categories (id, name, sort_order) VALUES (?,?,?)", c.ID, c.Name, c.SortOrder)
	return err
}

func (s *Store) DeleteCategory(id string) error {
	_, err := s.db.Exec("DELETE FROM categories WHERE id=?", id)
	return err
}

// --- Packages ---

func (s *Store) GetPackages() ([]models.Package, error) {
	rows, err := s.db.Query(`SELECT id,name,description,category_id,product_type,price,monthly_anchor,
		pricing_strategy,cores,ram_mb,disk_gb,os_type,vm_type,is_available,stock_limit,current_stock,tiers,created_at
		FROM packages ORDER BY price`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pkgs []models.Package
	for rows.Next() {
		var p models.Package
		var tiersJSON string
		var createdAt string
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.ProductType, &p.Price, &p.MonthlyAnchor,
			&p.PricingStrategy, &p.Cores, &p.RamMB, &p.DiskGB, &p.OSType, &p.VMType, &p.IsAvailable, &p.StockLimit, &p.CurrentStock, &tiersJSON, &createdAt)
		json.Unmarshal([]byte(tiersJSON), &p.Tiers)
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

func (s *Store) GetPackage(id string) (*models.Package, error) {
	var p models.Package
	var tiersJSON string
	var createdAt string
	err := s.db.QueryRow(`SELECT id,name,description,category_id,product_type,price,monthly_anchor,
		pricing_strategy,cores,ram_mb,disk_gb,os_type,vm_type,is_available,stock_limit,current_stock,tiers,created_at
		FROM packages WHERE id=?`, id).Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.ProductType, &p.Price, &p.MonthlyAnchor,
		&p.PricingStrategy, &p.Cores, &p.RamMB, &p.DiskGB, &p.OSType, &p.VMType, &p.IsAvailable, &p.StockLimit, &p.CurrentStock, &tiersJSON, &createdAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(tiersJSON), &p.Tiers)
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &p, nil
}

func (s *Store) SavePackage(p *models.Package) error {
	tiersJSON, _ := json.Marshal(p.Tiers)
	_, err := s.db.Exec(`INSERT OR REPLACE INTO packages
		(id,name,description,category_id,product_type,price,monthly_anchor,pricing_strategy,
		cores,ram_mb,disk_gb,os_type,vm_type,is_available,stock_limit,current_stock,tiers)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Description, p.CategoryID, p.ProductType, p.Price, p.MonthlyAnchor, p.PricingStrategy,
		p.Cores, p.RamMB, p.DiskGB, p.OSType, p.VMType, p.IsAvailable, p.StockLimit, p.CurrentStock, string(tiersJSON))
	return err
}

func (s *Store) DeletePackage(id string) error {
	_, err := s.db.Exec("DELETE FROM packages WHERE id=?", id)
	return err
}

// --- Orders ---

func (s *Store) GetOrders(limit int) ([]models.Order, error) {
	rows, err := s.db.Query(`SELECT id,email,whatsapp,package_id,duration_hours,coupon_code,
		original_price,discount_amt,total_price,status,payment_ref,proxmox_vm_id,proxmox_node,
		login_ip,login_user,login_pass,created_at,paid_at,expires_at
		FROM orders ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var paidAt, expiresAt sql.NullString
		rows.Scan(&o.ID, &o.Email, &o.WhatsApp, &o.PackageID, &o.DurationHours, &o.CouponCode,
			&o.OriginalPrice, &o.DiscountAmt, &o.TotalPrice, &o.Status, &o.PaymentRef, &o.ProxmoxVMID, &o.ProxmoxNode,
			&o.LoginIP, &o.LoginUser, &o.LoginPass, &o.CreatedAt, &paidAt, &expiresAt)
		if paidAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", paidAt.String)
			o.PaidAt = &t
		}
		if expiresAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", expiresAt.String)
			o.ExpiresAt = &t
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (s *Store) GetOrder(id string) (*models.Order, error) {
	var o models.Order
	var paidAt, expiresAt sql.NullString
	err := s.db.QueryRow(`SELECT id,email,whatsapp,package_id,duration_hours,coupon_code,
		original_price,discount_amt,total_price,status,payment_ref,proxmox_vm_id,proxmox_node,
		login_ip,login_user,login_pass,created_at,paid_at,expires_at
		FROM orders WHERE id=?`, id).Scan(&o.ID, &o.Email, &o.WhatsApp, &o.PackageID, &o.DurationHours, &o.CouponCode,
		&o.OriginalPrice, &o.DiscountAmt, &o.TotalPrice, &o.Status, &o.PaymentRef, &o.ProxmoxVMID, &o.ProxmoxNode,
		&o.LoginIP, &o.LoginUser, &o.LoginPass, &o.CreatedAt, &paidAt, &expiresAt)
	if err != nil {
		return nil, err
	}
	if paidAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", paidAt.String)
		o.PaidAt = &t
	}
	if expiresAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", expiresAt.String)
		o.ExpiresAt = &t
	}
	return &o, nil
}

func (s *Store) SaveOrder(o *models.Order) error {
	var paidAt, expiresAt interface{}
	if o.PaidAt != nil {
		paidAt = o.PaidAt.Format("2006-01-02 15:04:05")
	}
	if o.ExpiresAt != nil {
		expiresAt = o.ExpiresAt.Format("2006-01-02 15:04:05")
	}
	_, err := s.db.Exec(`INSERT OR REPLACE INTO orders
		(id,email,whatsapp,package_id,duration_hours,coupon_code,original_price,discount_amt,
		total_price,status,payment_ref,proxmox_vm_id,proxmox_node,login_ip,login_user,login_pass,paid_at,expires_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID, o.Email, o.WhatsApp, o.PackageID, o.DurationHours, o.CouponCode, o.OriginalPrice, o.DiscountAmt,
		o.TotalPrice, o.Status, o.PaymentRef, o.ProxmoxVMID, o.ProxmoxNode, o.LoginIP, o.LoginUser, o.LoginPass, paidAt, expiresAt)
	return err
}

func (s *Store) UpdateOrderStatus(id, status string) error {
	_, err := s.db.Exec("UPDATE orders SET status=? WHERE id=?", status, id)
	return err
}

// --- Coupons ---

func (s *Store) GetCoupons() ([]models.Coupon, error) {
	rows, err := s.db.Query(`SELECT id,code,discount_pct,discount_amt,max_uses,used_count,expires_at,is_active,created_at FROM coupons ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var coupons []models.Coupon
	for rows.Next() {
		var c models.Coupon
		var exp sql.NullString
		rows.Scan(&c.ID, &c.Code, &c.DiscountPct, &c.DiscountAmt, &c.MaxUses, &c.UsedCount, &exp, &c.IsActive, &c.CreatedAt)
		if exp.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", exp.String)
			c.ExpiresAt = &t
		}
		coupons = append(coupons, c)
	}
	return coupons, nil
}

func (s *Store) GetCoupon(code string) (*models.Coupon, error) {
	var c models.Coupon
	var exp sql.NullString
	err := s.db.QueryRow(`SELECT id,code,discount_pct,discount_amt,max_uses,used_count,expires_at,is_active,created_at FROM coupons WHERE code=?`, code).
		Scan(&c.ID, &c.Code, &c.DiscountPct, &c.DiscountAmt, &c.MaxUses, &c.UsedCount, &exp, &c.IsActive, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	if exp.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", exp.String)
		c.ExpiresAt = &t
	}
	return &c, nil
}

func (s *Store) SaveCoupon(c *models.Coupon) error {
	var exp interface{}
	if c.ExpiresAt != nil {
		exp = c.ExpiresAt.Format("2006-01-02 15:04:05")
	}
	_, err := s.db.Exec(`INSERT OR REPLACE INTO coupons (id,code,discount_pct,discount_amt,max_uses,used_count,expires_at,is_active)
		VALUES (?,?,?,?,?,?,?,?)`, c.ID, c.Code, c.DiscountPct, c.DiscountAmt, c.MaxUses, c.UsedCount, exp, c.IsActive)
	return err
}

func (s *Store) DeleteCoupon(id string) error {
	_, err := s.db.Exec("DELETE FROM coupons WHERE id=?", id)
	return err
}

func (s *Store) IncrementCouponUse(code string) error {
	_, err := s.db.Exec("UPDATE coupons SET used_count=used_count+1 WHERE code=?", code)
	return err
}

// --- Settings ---

func (s *Store) GetSetting(key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key=?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO settings (key,value) VALUES (?,?)", key, value)
	return err
}

func (s *Store) GetAllSettings() (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		m[k] = v
	}
	return m, nil
}

// --- Admin Auth ---

func (s *Store) GetAdminUser(username string) (*models.AdminUser, error) {
	var u models.AdminUser
	err := s.db.QueryRow("SELECT id,username,password,role FROM admin_users WHERE username=?", username).Scan(&u.ID, &u.Username, &u.Password, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpdateAdminPassword(username, hash string) error {
	_, err := s.db.Exec("UPDATE admin_users SET password=? WHERE username=?", hash, username)
	return err
}

// --- Stats for Dashboard ---

func (s *Store) GetStats() (map[string]int, error) {
	stats := map[string]int{}
	var v int
	s.db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&v); stats["total_orders"] = v
	s.db.QueryRow("SELECT COUNT(*) FROM orders WHERE status='active'").Scan(&v); stats["active_orders"] = v
	s.db.QueryRow("SELECT COUNT(*) FROM orders WHERE status='pending'").Scan(&v); stats["pending_orders"] = v
	s.db.QueryRow("SELECT COUNT(*) FROM packages WHERE is_available=1").Scan(&v); stats["active_packages"] = v
	s.db.QueryRow("SELECT COUNT(*) FROM coupons WHERE is_active=1").Scan(&v); stats["active_coupons"] = v
	s.db.QueryRow("SELECT COALESCE(SUM(total_price),0) FROM orders WHERE status IN ('paid','active')").Scan(&v); stats["total_revenue"] = v
	return stats, nil
}
