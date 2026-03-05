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

// Stats represents system statistics
type Stats struct {
	CPU    CPUStats    `json:"cpu"`
	Memory MemoryStats `json:"memory"`
	Disk   DiskStats   `json:"disk"`
	System SystemStats `json:"system"`
	Apps   AppsStats   `json:"apps"`
}

type CPUStats struct {
	Usage   float64   `json:"usage"`
	Cores   int       `json:"cores"`
	Model   string    `json:"model"`
	LoadAvg []float64 `json:"loadAvg"`
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
