package models

import "time"

// App represents a deployed application
type App struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	RepoURL    string            `json:"repo_url"`
	Branch     string            `json:"branch"`
	Port       int               `json:"port"`
	Domain     *string           `json:"domain"`
	SSLEnabled bool              `json:"ssl_enabled"`
	EnvVars    map[string]string `json:"env_vars"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	// Enriched fields from PM2 (not stored in DB)
	Status string  `json:"status,omitempty"`
	CPU    float64 `json:"cpu,omitempty"`
	Memory int64   `json:"memory,omitempty"`
}

// Database represents a managed PostgreSQL database
type Database struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	DBUser    string    `json:"db_user"`
	Password  string    `json:"-"` // never serialized
	CreatedAt time.Time `json:"created_at"`
}

// AuditEntry represents an audit log record
type AuditEntry struct {
	ID         int64     `json:"id"`
	Username   string    `json:"username"`
	IP         string    `json:"ip"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	DurationMs int       `json:"duration_ms"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// Pm2Process represents a PM2 managed process
type Pm2Process struct {
	Name   string  `json:"name"`
	PmID   int     `json:"pm_id"`
	Status string  `json:"status"`
	CPU    float64 `json:"cpu"`
	Memory int64   `json:"memory"`
	Uptime int64   `json:"uptime"`
}

// Stats represents system statistics (current snapshot)
type Stats struct {
	CPU       CPUStats       `json:"cpu"`
	Memory    MemoryStats    `json:"memory"`
	Disk      DiskStats      `json:"disk"`
	Network   NetworkStats   `json:"network"`
	DiskIO    DiskIOStats    `json:"diskIO"`
	System    SystemStats    `json:"system"`
	Apps      AppsStats      `json:"apps"`
	Processes []ProcessStats `json:"processes"`
}

// LiveStats is the full payload sent over WebSocket (current + history)
type LiveStats struct {
	Current *Stats       `json:"current"`
	History StatsHistory `json:"history"`
}

type CPUStats struct {
	Usage   float64      `json:"usage"`
	Cores   int          `json:"cores"`
	Model   string       `json:"model"`
	LoadAvg []float64    `json:"loadAvg"`
	PerCore []float64    `json:"perCore"`
}

type MemoryStats struct {
	Total   int64   `json:"total"`
	Used    int64   `json:"used"`
	Free    int64   `json:"free"`
	Percent float64 `json:"percent"`
}

type DiskStats struct {
	Total   int64   `json:"total"`
	Used    int64   `json:"used"`
	Percent float64 `json:"percent"`
}

type NetworkStats struct {
	RxBytesPerSec int64  `json:"rxBytesPerSec"`
	TxBytesPerSec int64  `json:"txBytesPerSec"`
	RxTotal       int64  `json:"rxTotal"`
	TxTotal       int64  `json:"txTotal"`
	Interface     string `json:"interface"`
}

type DiskIOStats struct {
	ReadBytesPerSec  int64  `json:"readBytesPerSec"`
	WriteBytesPerSec int64  `json:"writeBytesPerSec"`
	ReadTotal        int64  `json:"readTotal"`
	WriteTotal       int64  `json:"writeTotal"`
	Device           string `json:"device"`
}

type ProcessStats struct {
	PID     int     `json:"pid"`
	Name    string  `json:"name"`
	CPU     float64 `json:"cpu"`
	Memory  int64   `json:"memory"`
	MemPct  float64 `json:"memPct"`
	User    string  `json:"user"`
	Command string  `json:"command"`
}

type SystemStats struct {
	Uptime   string `json:"uptime"`
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
}

type AppsStats struct {
	Total   int         `json:"total"`
	Running int         `json:"running"`
	Stopped int         `json:"stopped"`
	List    interface{} `json:"list"`
}

// StatsHistory holds ring buffer data points for sparkline charts
type StatsHistory struct {
	Timestamps []int64   `json:"timestamps"` // unix seconds
	CPU        []float64 `json:"cpu"`
	Memory     []float64 `json:"memory"`
	DiskRead   []int64   `json:"diskRead"`
	DiskWrite  []int64   `json:"diskWrite"`
	NetRx      []int64   `json:"netRx"`
	NetTx      []int64   `json:"netTx"`
}

// ExecResult holds the result of a command execution
type ExecResult struct {
	Stdout string
	Stderr string
	Code   int
}

// ApiResponse is the standard JSON envelope
type ApiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// FileEntry represents a file/directory listing item
type FileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "file" or "dir"
	Path string `json:"path"`
}

// RedisInfo represents Redis status
type RedisInfo struct {
	Installed  bool        `json:"installed"`
	Running    bool        `json:"running"`
	Connection interface{} `json:"connection"`
}

type RedisConnection struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	URL    string `json:"url"`
	EnvVar string `json:"env_var"`
}
