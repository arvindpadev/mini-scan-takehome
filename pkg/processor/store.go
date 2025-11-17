package processor

import (
	"context"
)

type StorableScan struct {
	ip        string
	port      uint32
	service   string
	timestamp int64
	data      string
}

type Store interface {
	StoreScan(ctx context.Context, scan *StorableScan) error
}
