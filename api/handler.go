package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"vps-rental/models"
	"vps-rental/proxmox"
	"vps-rental/store"
)

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *Handler) errResp(w http.ResponseWriter, code int, msg string) {
	h.respond(w, code, map[string]string{"error": msg})
}

func genID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RegisterRoutes registers all API routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Public API
	mux.HandleFunc("/api/v1/categories", h.handleCategories)
	mux.HandleFunc("/api/v1/packages", h.handlePackages)
	mux.HandleFunc("/api/v1/orders", h.handleOrders)
	mux.HandleFunc("/api/v1/coupons/validate", h.handleCouponValidate)
	mux.HandleFunc("/api/v1/track/", h.handleTrackOrder)

	// Admin API
	mux.HandleFunc("/admin/api/login", h.adminLogin)
	mux.HandleFunc("/admin/api/logout", h.adminLogout)
	mux.HandleFunc("/admin/api/stats", h.adminAuth(h.adminStats))
	mux.HandleFunc("/admin/api/packages", h.adminAuth(h.adminPackages))
	mux.HandleFunc("/admin/api/packages/", h.adminAuth(h.adminPackageItem))
	mux.HandleFunc("/admin/api/categories", h.adminAuth(h.adminCategories))
	mux.HandleFunc("/admin/api/orders", h.adminAuth(h.adminOrders))
	mux.HandleFunc("/admin/api/orders/", h.adminAuth(h.adminOrderItem))
	mux.HandleFunc("/admin/api/coupons", h.adminAuth(h.adminCoupons))
	mux.HandleFunc("/admin/api/coupons/", h.adminAuth(h.adminCouponItem))
	mux.HandleFunc("/admin/api/settings", h.adminAuth(h.adminSettings))
	mux.HandleFunc("/admin/api/proxmox/status", h.adminAuth(h.proxmoxStatus))
	mux.HandleFunc("/admin/api/proxmox/vms", h.adminAuth(h.proxmoxVMs))
	mux.HandleFunc("/admin/api/proxmox/create", h.adminAuth(h.proxmoxCreate))
	mux.HandleFunc("/admin/api/proxmox/action/", h.adminAuth(h.proxmoxAction))
}

// ============= PUBLIC API (v1) =============

func (h *Handler) handleCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.store.GetCategories()
	if err != nil {
		h.errResp(w, 500, err.Error())
		return
	}
	if cats == nil {
		cats = []models.Category{}
	}
	h.respond(w, 200, cats)
}

func (h *Handler) handlePackages(w http.ResponseWriter, r *http.Request) {
	pkgs, err := h.store.GetPackages()
	if err != nil {
		h.errResp(w, 500, err.Error())
		return
	}
	if pkgs == nil {
		pkgs = []models.Package{}
	}
	h.respond(w, 200, pkgs)
}

func (h *Handler) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		h.createOrder(w, r)
	case "GET":
		h.listOrders(w, r)
	default:
		h.errResp(w, 405, "method not allowed")
	}
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email         string `json:"email"`
		WhatsApp      string `json:"whatsapp"`
		PackageID     string `json:"packageId"`
		DurationHours int    `json:"durationHours"`
		CouponCode    string `json:"couponCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errResp(w, 400, "invalid JSON: "+err.Error())
		return
	}
	if req.Email == "" || req.WhatsApp == "" || req.PackageID == "" || req.DurationHours == 0 {
		h.errResp(w, 400, "email, whatsapp, packageId, durationHours required")
		return
	}

	pkg, err := h.store.GetPackage(req.PackageID)
	if err != nil {
		h.errResp(w, 404, "package not found")
		return
	}
	if !pkg.IsAvailable {
		h.errResp(w, 400, "package not available")
		return
	}
	if pkg.StockLimit > 0 && pkg.CurrentStock >= pkg.StockLimit {
		h.errResp(w, 400, "stok untuk paket ini sementara penuh. Silakan pilih paket lain atau hubungi admin.")
		return
	}

	// find tier price
	var tierPrice float64
	for _, t := range pkg.Tiers {
		if t.DurationHours == req.DurationHours {
			tierPrice = t.Price
			break
		}
	}
	if tierPrice == 0 {
		// fallback: proportional from monthly
		tierPrice = float64(pkg.MonthlyAnchor) * float64(req.DurationHours) / 720.0
	}

	originalPrice := tierPrice
	discountAmt := 0.0

	// coupon
	if req.CouponCode != "" {
		coupon, err := h.store.GetCoupon(req.CouponCode)
		if err == nil && coupon.IsActive {
			expired := coupon.ExpiresAt != nil && coupon.ExpiresAt.Before(time.Now())
			maxReached := coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses
			if !expired && !maxReached {
				if coupon.DiscountPct > 0 {
					discountAmt = originalPrice * coupon.DiscountPct / 100
				} else if coupon.DiscountAmt > 0 {
					discountAmt = coupon.DiscountAmt
				}
			}
		}
	}

	total := originalPrice - discountAmt
	if total < 0 {
		total = 0
	}

	order := &models.Order{
		ID:            genID(),
		Email:         req.Email,
		WhatsApp:      req.WhatsApp,
		PackageID:     req.PackageID,
		DurationHours: req.DurationHours,
		CouponCode:    req.CouponCode,
		OriginalPrice: originalPrice,
		DiscountAmt:   discountAmt,
		TotalPrice:    total,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := h.store.SaveOrder(order); err != nil {
		h.errResp(w, 500, "failed to create order: "+err.Error())
		return
	}

	h.respond(w, 201, order)
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	orders, err := h.store.GetOrders(limit)
	if err != nil {
		h.errResp(w, 500, err.Error())
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}
	h.respond(w, 200, orders)
}

func (h *Handler) handleCouponValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.errResp(w, 405, "POST required")
		return
	}
	var req struct {
		Code    string `json:"code"`
		Price   float64 `json:"price"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	coupon, err := h.store.GetCoupon(req.Code)
	if err != nil {
		h.errResp(w, 404, "kupon tidak ditemukan")
		return
	}
	if !coupon.IsActive {
		h.errResp(w, 400, "kupon tidak aktif")
		return
	}
	if coupon.ExpiresAt != nil && coupon.ExpiresAt.Before(time.Now()) {
		h.errResp(w, 400, "kupon sudah kadaluarsa")
		return
	}
	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		h.errResp(w, 400, "kupon sudah mencapai batas penggunaan")
		return
	}

	discount := 0.0
	if coupon.DiscountPct > 0 {
		discount = req.Price * coupon.DiscountPct / 100
	} else {
		discount = coupon.DiscountAmt
	}
	h.respond(w, 200, map[string]interface{}{
		"valid":      true,
		"discount":   discount,
		"finalPrice": req.Price - discount,
	})
}

func (h *Handler) handleTrackOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/track/")
	if id == "" {
		h.errResp(w, 400, "order ID required")
		return
	}
	order, err := h.store.GetOrder(id)
	if err != nil {
		h.errResp(w, 404, "pesanan tidak ditemukan")
		return
	}
	h.respond(w, 200, map[string]interface{}{
		"id":         order.ID,
		"status":     order.Status,
		"packageId":  order.PackageID,
		"duration":   order.DurationHours,
		"total":      order.TotalPrice,
		"createdAt":  order.CreatedAt,
		"loginIp":    order.LoginIP,
		"loginUser":  order.LoginUser,
	})
}

// ============= ADMIN API =============

// --- Admin Auth ---

func (h *Handler) adminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.errResp(w, 405, "POST required")
		return
	}
	var req struct{ Username, Password string }
	json.NewDecoder(r.Body).Decode(&req)

	user, err := h.store.GetAdminUser(req.Username)
	if err != nil {
		h.errResp(w, 401, "username atau password salah")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.errResp(w, 401, "username atau password salah")
		return
	}

	// simple session token
	token := genID()
	h.store.SetSetting("session_"+token, user.Username)
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
	})
	h.respond(w, 200, map[string]string{"token": token, "role": user.Role})
}

func (h *Handler) adminLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("admin_token")
	if err == nil && cookie.Value != "" {
		h.store.SetSetting("session_"+cookie.Value, "")
	}
	http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: "", Path: "/", MaxAge: -1})
	h.respond(w, 200, map[string]string{"status": "logged out"})
}

func (h *Handler) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("admin_token")
		if err != nil || cookie.Value == "" {
			h.errResp(w, 401, "unauthorized")
			return
		}
		user, _ := h.store.GetSetting("session_" + cookie.Value)
		if user == "" {
			h.errResp(w, 401, "session expired")
			return
		}
		next(w, r)
	}
}

// --- Admin Stats ---

func (h *Handler) adminStats(w http.ResponseWriter, r *http.Request) {
	stats, _ := h.store.GetStats()
	h.respond(w, 200, stats)
}

// --- Admin Packages ---

func (h *Handler) adminPackages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		pkgs, err := h.store.GetPackages()
		if err != nil {
			h.errResp(w, 500, err.Error())
			return
		}
		if pkgs == nil {
			pkgs = []models.Package{}
		}
		h.respond(w, 200, pkgs)
	case "POST":
		var p models.Package
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			h.errResp(w, 400, err.Error())
			return
		}
		if p.ID == "" {
			p.ID = genID()
		}
		p.CreatedAt = time.Now()
		if err := h.store.SavePackage(&p); err != nil {
			h.errResp(w, 500, err.Error())
			return
		}
		h.respond(w, 201, p)
	default:
		h.errResp(w, 405, "method not allowed")
	}
}

func (h *Handler) adminPackageItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/api/packages/")
	switch r.Method {
	case "GET":
		p, err := h.store.GetPackage(id)
		if err != nil {
			h.errResp(w, 404, "not found")
			return
		}
		h.respond(w, 200, p)
	case "PUT":
		var p models.Package
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			h.errResp(w, 400, err.Error())
			return
		}
		p.ID = id
		h.store.SavePackage(&p)
		h.respond(w, 200, p)
	case "DELETE":
		h.store.DeletePackage(id)
		h.respond(w, 200, map[string]string{"status": "deleted"})
	default:
		h.errResp(w, 405, "method not allowed")
	}
}

// --- Admin Categories ---

func (h *Handler) adminCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		cats, _ := h.store.GetCategories()
		if cats == nil {
			cats = []models.Category{}
		}
		h.respond(w, 200, cats)
	case "POST":
		var c models.Category
		json.NewDecoder(r.Body).Decode(&c)
		if c.ID == "" {
			c.ID = genID()
		}
		h.store.SaveCategory(&c)
		h.respond(w, 201, c)
	case "DELETE":
		id := r.URL.Query().Get("id")
		h.store.DeleteCategory(id)
		h.respond(w, 200, map[string]string{"status": "deleted"})
	default:
		h.errResp(w, 405, "method not allowed")
	}
}

// --- Admin Orders ---

func (h *Handler) adminOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil {
				limit = v
			}
		}
		orders, _ := h.store.GetOrders(limit)
		if orders == nil {
			orders = []models.Order{}
		}
		h.respond(w, 200, orders)
	case "POST":
		// update order status
		var req struct{ ID, Status string }
		json.NewDecoder(r.Body).Decode(&req)
		if err := h.store.UpdateOrderStatus(req.ID, req.Status); err != nil {
			h.errResp(w, 500, err.Error())
			return
		}
		h.respond(w, 200, map[string]string{"status": "updated"})
	default:
		h.errResp(w, 405, "method not allowed")
	}
}

func (h *Handler) adminOrderItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/api/orders/")
	order, err := h.store.GetOrder(id)
	if err != nil {
		h.errResp(w, 404, "not found")
		return
	}
	if r.Method == "PUT" {
		var updated models.Order
		json.NewDecoder(r.Body).Decode(&updated)
		updated.ID = id
		h.store.SaveOrder(&updated)
		h.respond(w, 200, updated)
		return
	}
	h.respond(w, 200, order)
}

// --- Admin Coupons ---

func (h *Handler) adminCoupons(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		coupons, _ := h.store.GetCoupons()
		if coupons == nil {
			coupons = []models.Coupon{}
		}
		h.respond(w, 200, coupons)
	case "POST":
		var c models.Coupon
		json.NewDecoder(r.Body).Decode(&c)
		if c.ID == "" {
			c.ID = genID()
		}
		c.CreatedAt = time.Now()
		h.store.SaveCoupon(&c)
		h.respond(w, 201, c)
	default:
		h.errResp(w, 405, "method not allowed")
	}
}

func (h *Handler) adminCouponItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/api/coupons/")
	if r.Method == "DELETE" {
		h.store.DeleteCoupon(id)
		h.respond(w, 200, map[string]string{"status": "deleted"})
		return
	}
	h.errResp(w, 405, "method not allowed")
}

// --- Admin Settings ---

func (h *Handler) adminSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		settings, _ := h.store.GetAllSettings()
		// don't expose passwords
		if _, ok := settings["proxmox_password"]; ok {
			settings["proxmox_password"] = "***"
		}
		h.respond(w, 200, settings)
	case "POST":
		var settings map[string]string
		json.NewDecoder(r.Body).Decode(&settings)
		for k, v := range settings {
			if k == "proxmox_password" && v == "***" {
				continue
			}
			h.store.SetSetting(k, v)
		}
		h.respond(w, 200, map[string]string{"status": "saved"})
	default:
		h.errResp(w, 405, "method not allowed")
	}
}

// --- Proxmox ---

func (h *Handler) getProxmoxClient() *proxmox.Client {
	settings, _ := h.store.GetAllSettings()
	cfg := proxmox.ParseProxmoxConfigFromSettings(settings)
	return proxmox.NewClient(cfg)
}

func (h *Handler) proxmoxStatus(w http.ResponseWriter, r *http.Request) {
	client := h.getProxmoxClient()
	connected := client.IsConnected()
	resp := map[string]interface{}{
		"connected": connected,
	}
	if connected {
		nodeStatus, err := client.GetNodeStatus()
		if err == nil {
			if data, ok := nodeStatus["data"].(map[string]interface{}); ok {
				resp["node"] = data
			}
		}
	}
	h.respond(w, 200, resp)
}

func (h *Handler) proxmoxVMs(w http.ResponseWriter, r *http.Request) {
	client := h.getProxmoxClient()
	vms, err := client.GetVMs()
	if err != nil {
		h.errResp(w, 500, "Proxmox: "+err.Error())
		return
	}
	if vms == nil {
		vms = []models.ProxmoxVM{}
	}
	// also get QEMU VMs
	qemu, _ := client.GetQEMUVMs()
	vms = append(vms, qemu...)
	h.respond(w, 200, vms)
}

func (h *Handler) proxmoxCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.errResp(w, 405, "POST required")
		return
	}
	var req struct {
		VMID     int    `json:"vmid"`
		Hostname string `json:"hostname"`
		Cores    int    `json:"cores"`
		RamMB    int    `json:"ramMb"`
		DiskGB   int    `json:"diskGb"`
		Template string `json:"template"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	client := h.getProxmoxClient()
	if req.VMID == 0 {
		id, err := client.NextVMID()
		if err != nil {
			h.errResp(w, 500, "failed to get next VMID: "+err.Error())
			return
		}
		req.VMID = id
	}

	// generate password if not provided
	if req.Password == "" {
		req.Password = genPassword(12)
	}

	if err := client.CreateLXC(req.VMID, req.Hostname, req.Cores, req.RamMB, req.DiskGB, req.Template, req.Password); err != nil {
		h.errResp(w, 500, "failed to create LXC: "+err.Error())
		return
	}
	h.respond(w, 201, map[string]interface{}{
		"vmid":     req.VMID,
		"hostname": req.Hostname,
		"password": req.Password,
		"status":   "created",
	})
}

func (h *Handler) proxmoxAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/admin/api/proxmox/action/"), "/")
	if len(parts) < 2 {
		h.errResp(w, 400, "format: /action/{vmid}/{start|stop|delete}")
		return
	}
	vmid, _ := strconv.Atoi(parts[0])
	action := parts[1]
	client := h.getProxmoxClient()

	switch action {
	case "start":
		if err := client.StartVM(vmid); err != nil {
			h.errResp(w, 500, err.Error())
			return
		}
	case "stop":
		if err := client.StopVM(vmid); err != nil {
			h.errResp(w, 500, err.Error())
			return
		}
	case "delete":
		if err := client.DeleteVM(vmid); err != nil {
			h.errResp(w, 500, err.Error())
			return
		}
	default:
		h.errResp(w, 400, "unknown action: "+action)
		return
	}

	h.respond(w, 200, map[string]interface{}{
		"vmid":   vmid,
		"action": action,
		"status": "done",
	})
}

func genPassword(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:n]
}

func init() {
	log.Println("API handler ready")
}
