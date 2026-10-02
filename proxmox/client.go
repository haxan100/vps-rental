package proxmox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"vps-rental/models"
)

type Client struct {
	host     string
	port     int
	user     string
	password string
	realm    string
	node     string
	token    string
	client   *http.Client
}

func NewClient(cfg models.ProxmoxConfig) *Client {
	return &Client{
		host:     cfg.Host,
		port:     cfg.Port,
		user:     cfg.User,
		password: cfg.Password,
		realm:    cfg.Realm,
		node:     cfg.Node,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (c *Client) baseURL() string {
	return fmt.Sprintf("https://%s:%d/api2/json", c.host, c.port)
}

func (c *Client) authenticate() error {
	form := url.Values{}
	form.Set("username", c.user)
	form.Set("password", c.password)
	form.Set("realm", c.realm)

	resp, err := c.client.PostForm(c.baseURL()+"/access/ticket", form)
	if err != nil {
		return fmt.Errorf("proxmox auth: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Ticket string `json:"ticket"`
			CSRF   string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Data.Ticket == "" {
		return fmt.Errorf("proxmox auth failed: no ticket returned")
	}
	c.token = result.Data.Ticket
	return nil
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	if c.token == "" {
		if err := c.authenticate(); err != nil {
			return nil, err
		}
	}

	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, _ := http.NewRequest(method, c.baseURL()+path, bodyReader)
	req.Header.Set("Cookie", "PVEAuthCookie="+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.token = "" // re-auth next time
		return nil, fmt.Errorf("proxmox request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		c.token = ""
		return nil, fmt.Errorf("proxmox error %d: %s", resp.StatusCode, string(data)[:min(200, len(data))])
	}
	return data, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetVMs returns all VMs/CTs on the node
func (c *Client) GetVMs() ([]models.ProxmoxVM, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/nodes/%s/lxc", c.node), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []struct {
			VMID   int    `json:"vmid"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Cores  int    `json:"cpus"`
			MaxMem int64  `json:"maxmem"`
			MaxDisk int64 `json:"maxdisk"`
			Uptime int64  `json:"uptime"`
		} `json:"data"`
	}
	json.Unmarshal(data, &result)

	var vms []models.ProxmoxVM
	for _, v := range result.Data {
		vms = append(vms, models.ProxmoxVM{
			VMID:   v.VMID,
			Name:   v.Name,
			Status: v.Status,
			Cores:  v.Cores,
			RamMB:  int(v.MaxMem / 1024 / 1024),
			DiskGB: int(v.MaxDisk / 1024 / 1024 / 1024),
			VMType: "lxc",
			Uptime: v.Uptime,
			Node:   c.node,
		})
	}
	return vms, nil
}

// GetQEMUVMs returns QEMU VMs
func (c *Client) GetQEMUVMs() ([]models.ProxmoxVM, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/nodes/%s/qemu", c.node), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []struct {
			VMID   int    `json:"vmid"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Cores  int    `json:"cpus"`
			MaxMem int64  `json:"maxmem"`
			MaxDisk int64 `json:"maxdisk"`
			OSType string `json:"ostype"`
			Uptime int64  `json:"uptime"`
		} `json:"data"`
	}
	json.Unmarshal(data, &result)

	var vms []models.ProxmoxVM
	for _, v := range result.Data {
		vms = append(vms, models.ProxmoxVM{
			VMID:   v.VMID,
			Name:   v.Name,
			Status: v.Status,
			Cores:  v.Cores,
			RamMB:  int(v.MaxMem / 1024 / 1024),
			DiskGB: int(v.MaxDisk / 1024 / 1024 / 1024),
			OSType: v.OSType,
			VMType: "qemu",
			Uptime: v.Uptime,
			Node:   c.node,
		})
	}
	return vms, nil
}

// CreateLXC creates a new LXC container
func (c *Client) CreateLXC(vmid int, hostname string, cores, ramMB, diskGB int, ostemplate string, password string) error {
	body := map[string]interface{}{
		"vmid":     vmid,
		"hostname": hostname,
		"cores":    cores,
		"memory":   ramMB,
		"rootfs":   fmt.Sprintf("local-lvm:%d", diskGB),
		"ostemplate": ostemplate,
		"password": password,
		"net0":    "name=eth0,bridge=vmbr0,ip=dhcp",
		"start":   1,
		"unprivileged": 1,
	}
	_, err := c.doRequest("POST", fmt.Sprintf("/nodes/%s/lxc", c.node), body)
	return err
}

// StartVM starts a VM/CT
func (c *Client) StartVM(vmid int) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/nodes/%s/lxc/%d/status/start", c.node, vmid), nil)
	return err
}

// StopVM stops a VM/CT
func (c *Client) StopVM(vmid int) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/nodes/%s/lxc/%d/status/stop", c.node, vmid), nil)
	return err
}

// DeleteVM removes a VM/CT
func (c *Client) DeleteVM(vmid int) error {
	// stop first if running
	c.StopVM(vmid)
	time.Sleep(2 * time.Second)
	_, err := c.doRequest("DELETE", fmt.Sprintf("/nodes/%s/lxc/%d", c.node, vmid), nil)
	return err
}

// GetVMStatus gets status of a single VM
func (c *Client) GetVMStatus(vmid int) (map[string]interface{}, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/nodes/%s/lxc/%d/status/current", c.node, vmid), nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}

// NextVMID gets the next available VMID
func (c *Client) NextVMID() (int, error) {
	data, err := c.doRequest("GET", "/cluster/nextid", nil)
	if err != nil {
		return 0, err
	}
	var result struct {
		Data string `json:"data"`
	}
	json.Unmarshal(data, &result)
	var id int
	fmt.Sscanf(result.Data, "%d", &id)
	return id, nil
}

// GetNodeStatus gets node resource usage
func (c *Client) GetNodeStatus() (map[string]interface{}, error) {
	data, err := c.doRequest("GET", fmt.Sprintf("/nodes/%s/status", c.node), nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}

// IsConnected checks if Proxmox is reachable
func (c *Client) IsConnected() bool {
	err := c.authenticate()
	if err != nil {
		return false
	}
	_, err = c.doRequest("GET", fmt.Sprintf("/nodes/%s/status", c.node), nil)
	return err == nil
}

// ParseProxmoxConfigFromSettings creates a Client from settings map
func ParseProxmoxConfigFromSettings(settings map[string]string) models.ProxmoxConfig {
	cfg := models.ProxmoxConfig{
		Host:   "127.0.0.1",
		Port:   8006,
		User:   "root",
		Realm:  "pam",
		Node:   "pve",
		SSHPort: 22,
	}
	if v, ok := settings["proxmox_host"]; ok && v != "" {
		cfg.Host = v
	}
	if v, ok := settings["proxmox_port"]; ok && v != "" {
		fmt.Sscanf(v, "%d", &cfg.Port)
	}
	if v, ok := settings["proxmox_user"]; ok && v != "" {
		cfg.User = v
	}
	if v, ok := settings["proxmox_password"]; ok {
		cfg.Password = v
	}
	if v, ok := settings["proxmox_realm"]; ok && v != "" {
		cfg.Realm = v
	}
	if v, ok := settings["proxmox_node"]; ok && v != "" {
		cfg.Node = v
	}
	if v, ok := settings["proxmox_ssh_port"]; ok && v != "" {
		fmt.Sscanf(v, "%d", &cfg.SSHPort)
	}
	return cfg
}
