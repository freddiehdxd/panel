package services

import (
	"context"
	"fmt"
)

// PortAllocator manages port allocation for apps
type PortAllocator struct {
	db       *DB
	portStart int
	portEnd   int
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(db *DB, portStart, portEnd int) *PortAllocator {
	return &PortAllocator{
		db:       db,
		portStart: portStart,
		portEnd:   portEnd,
	}
}

// Allocate finds and returns the next free port
func (pa *PortAllocator) Allocate(ctx context.Context) (int, error) {
	// Get all used ports
	rows, err := pa.db.Query(ctx, "SELECT port FROM apps ORDER BY port")
	if err != nil {
		return 0, fmt.Errorf("query ports: %w", err)
	}
	defer rows.Close()

	usedPorts := make(map[int]bool)
	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return 0, fmt.Errorf("scan port: %w", err)
		}
		usedPorts[port] = true
	}

	// Find first free port in range
	for port := pa.portStart; port <= pa.portEnd; port++ {
		if !usedPorts[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free ports available in range %d-%d", pa.portStart, pa.portEnd)
}
