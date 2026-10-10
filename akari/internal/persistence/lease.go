package persistence

import (
	"context"
	"os"
	"path/filepath"
	"sync"
)

// Acquire holds an installation lease until release or process death. It is separate from save locks.
func (file *File) Acquire(ctx context.Context) (func(), error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	lease, err := os.OpenFile(filepath.Join(file.directory, "process.lock"), os.O_CREATE|os.O_RDWR, privateFile)
	if err != nil {
		return nil, err
	}

	err = acquire(ctx, lease)
	if err != nil {
		_ = lease.Close()

		return nil, err
	}

	once := new(sync.Once)

	return func() { once.Do(func() { _ = lease.Close() }) }, nil
}
