package models

import "time"

// Category groups packages
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

// Package is a VPS/RDP product
type Package struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	CategoryID      string    `json:"categoryId"`
	ProductType     string    `json:"productType"` // PROXMOX_VPS, RDP_WINDOWS, RADIO_STREAMING
	Price           int       `json:"price"`       // monthly anchor (Rp)
	MonthlyAnchor   int       `json:"monthlyAnchor"`
	PricingStrategy string    `json:"pricingStrategy"`
	Cores           int       `json:"cores"`
	RamMB           int       `json:"ramMb"`
	DiskGB          int       `json:"diskGb"`
	OSType          string    `json:"osType"`
	VMType          string    `json:"vmType"` // lxc or kvm
	IsAvailable     bool      `json:"isAvailable"`
	StockLimit      int       `json:"stockLimit"` // 0 = unlimited
	CurrentStock    int       `json:"currentStock"`
	Tiers           []Tier    `json:"tiers"`
	CreatedAt       time.Time `json:"createdAt"`
}

// Tier is a duration pricing tier
type Tier struct {
	DurationHours int     `json:"durationHours"`
	Label         string  `json:"label"`
	Price         float64 `json:"price"`
	HourlyRate    float64 `json:"hourlyRate"`
	DiscountPct   float64 `json:"discountPct"`
	Badge         string  `json:"badge"`
}

// Order is a customer order
type Order struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	WhatsApp      string    `json:"whatsapp"`
	PackageID     string    `json:"packageId"`
	DurationHours int       `json:"durationHours"`
	CouponCode    string    `json:"couponCode"`
	OriginalPrice float64   `json:"originalPrice"`
	DiscountAmt   float64   `json:"discountAmt"`
	TotalPrice    float64   `json:"totalPrice"`
	Status        string    `json:"status"` // pending, paid, provisioning, active, expired, cancelled
	PaymentRef    string    `json:"paymentRef"`
	ProxmoxVMID   int       `json:"proxmoxVmId"`
	ProxmoxNode   string    `json:"proxmoxNode"`
	LoginIP       string    `json:"loginIp"`
	LoginUser     string    `json:"loginUser"`
	LoginPass     string    `json:"loginPass"`
	CreatedAt     time.Time `json:"createdAt"`
	PaidAt        *time.Time `json:"paidAt"`
	ExpiresAt     *time.Time `json:"expiresAt"`
}

// Coupon for discounts
type Coupon struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	DiscountPct float64  `json:"discountPct"` // percentage
	DiscountAmt float64  `json:"discountAmt"` // or fixed amount
	MaxUses    int       `json:"maxUses"`
	UsedCount  int       `json:"usedCount"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
}

// AdminUser for admin panel auth
type AdminUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"` // bcrypt hash
	Role     string `json:"role"`     // admin, viewer
}

// Setting key-value store
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ProxmoxConfig stored in settings
type ProxmoxConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Realm    string `json:"realm"` // pam or pve
	Node     string `json:"node"`
	SSHPort  int    `json:"sshPort"`
}

// ProxmoxVM represents a VM/CT in Proxmox
type ProxmoxVM struct {
	VMID    int    `json:"vmid"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Cores   int    `json:"cores"`
	RamMB   int    `json:"maxmem"` // bytes in API, convert
	DiskGB  int    `json:"maxdisk"` // bytes in API, convert
	OSType  string `json:"ostype"`
	VMType  string `json:"type"` // qemu or lxc
	Uptime  int64  `json:"uptime"`
	Node    string `json:"node"`
}
