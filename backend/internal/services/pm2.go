package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"panel-backend/internal/models"
)

// PM2 manages PM2 process operations
type PM2 struct {
	exec *Executor
}

// NewPM2 creates a new PM2 service
func NewPM2(exec *Executor) *PM2 {
	return &PM2{exec: exec}
}

// pm2RawProcess represents the raw JSON structure from pm2 jlist
type pm2RawProcess struct {
	Name  string `json:"name"`
	PmID  int    `json:"pm_id"`
	Pm2Env struct {
		Status  string `json:"status"`
		PmUptime int64  `json:"pm_uptime"`
	} `json:"pm2_env"`
	Monit struct {
		CPU    float64 `json:"cpu"`
		Memory int64   `json:"memory"`
	} `json:"monit"`
}

// List returns all PM2 processes
func (p *PM2) List() ([]models.Pm2Process, error) {
	result, err := p.exec.RunBin("pm2", "jlist")
	if err != nil {
		return nil, fmt.Errorf("pm2 jlist: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("pm2 jlist failed: %s", result.Stderr)
	}

	// pm2 jlist outputs JSON to stdout
	stdout := strings.TrimSpace(result.Stdout)
	if stdout == "" || stdout == "[]" {
		return []models.Pm2Process{}, nil
	}

	var raw []pm2RawProcess
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, fmt.Errorf("parse pm2 jlist: %w", err)
	}

	processes := make([]models.Pm2Process, 0, len(raw))
	for _, r := range raw {
		processes = append(processes, models.Pm2Process{
			Name:   r.Name,
			PmID:   r.PmID,
			Status: r.Pm2Env.Status,
			CPU:    r.Monit.CPU,
			Memory: r.Monit.Memory,
			Uptime: r.Pm2Env.PmUptime,
		})
	}

	return processes, nil
}

// Action performs a PM2 action (start, stop, restart, delete) on an app
func (p *PM2) Action(action, appName string) (*models.ExecResult, error) {
	validActions := map[string]bool{
		"start": true, "stop": true, "restart": true, "delete": true,
	}
	if !validActions[action] {
		return nil, fmt.Errorf("invalid pm2 action: %s", action)
	}

	return p.exec.RunBin("pm2", action, appName)
}

// Logs retrieves PM2 logs for an app
func (p *PM2) Logs(appName string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}

	result, err := p.exec.RunBin("pm2", "logs", appName,
		"--lines", fmt.Sprintf("%d", lines), "--nostream")
	if err != nil {
		return "", fmt.Errorf("pm2 logs: %w", err)
	}

	// Combine stdout and stderr (PM2 logs outputs to both)
	output := result.Stdout
	if result.Stderr != "" {
		output = output + "\n" + result.Stderr
	}

	return strings.TrimSpace(output), nil
}
