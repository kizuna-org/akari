// Package persistence provides a local administrative store without choosing a production DB.
package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kizuna-org/akari/internal/continuity"
)

const (
	beforeRename     = "beforeRename"
	maxBytes         = 16 << 20
	privateDirectory = 0o700
	privateFile      = 0o600
	lockWait         = 100 * time.Millisecond
	lockPoll         = time.Millisecond
)

// File uses an exclusive OS lock, revision CAS, atomic rename and file/directory fsync.
// The directory must belong to this installation, on a local filesystem supporting these operations.
type File struct {
	directory string
	hook      func(string) error
}

func New(directory string) (*File, error) {
	if directory == "" {
		return nil, continuity.ErrConfig
	}

	err := os.MkdirAll(directory, privateDirectory)
	if err != nil {
		return nil, err
	}

	return &File{directory: directory, hook: nil}, nil
}

func (file *File) Load(ctx context.Context) (continuity.Frame, error) {
	err := ctx.Err()
	if err != nil {
		return continuity.Frame{}, err
	}

	unlock, err := file.lock(ctx)
	if err != nil {
		return continuity.Frame{}, err
	}

	defer unlock()

	return file.read()
}

func (file *File) Save(ctx context.Context, expected uint64, next continuity.Frame) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	unlock, err := file.lock(ctx)
	if err != nil {
		return err
	}

	defer unlock()

	current, err := file.read()
	if err != nil && !errors.Is(err, continuity.ErrAbsent) {
		return err
	}

	if !matching(current, expected, next) {
		return continuity.ErrConflict
	}

	data, err := json.Marshal(next)
	if err != nil {
		return err
	}

	if len(data) > maxBytes {
		return continuity.ErrState
	}

	return file.replace(ctx, data)
}

func (file *File) lock(ctx context.Context) (func(), error) {
	lock, err := os.OpenFile(filepath.Join(file.directory, "owner.lock"), os.O_CREATE|os.O_RDWR, privateFile)
	if err != nil {
		return nil, err
	}

	err = acquire(ctx, lock)
	if err != nil {
		_ = lock.Close()

		return nil, err
	}

	return func() { _ = lock.Close() }, nil
}

// Bounded polling also tolerates descriptors briefly inherited during another goroutine's fork/exec.
func acquire(ctx context.Context, lock *os.File) error {
	ctx, cancel := context.WithTimeout(ctx, lockWait)
	defer cancel()

	for {
		err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}

		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return err
		}

		timer := time.NewTimer(lockPoll)
		select {
		case <-ctx.Done():
			_ = timer.Stop()

			return errors.Join(continuity.ErrBusy, ctx.Err())
		case <-timer.C:
		}
	}
}

func matching(current continuity.Frame, expected uint64, next continuity.Frame) bool {
	return current.Sequence == expected && next.Sequence == expected+1 &&
		(expected == 0 || next.Identity == current.Identity)
}

func (file *File) read() (continuity.Frame, error) {
	input, err := os.Open(filepath.Join(file.directory, "state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return continuity.Frame{}, continuity.ErrAbsent
	}

	if err != nil {
		return continuity.Frame{}, err
	}

	defer func() { _ = input.Close() }()

	data, err := io.ReadAll(io.LimitReader(input, maxBytes+1))
	if err != nil {
		return continuity.Frame{}, err
	}

	if len(data) > maxBytes {
		return continuity.Frame{}, continuity.ErrState
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var frame continuity.Frame

	err = decoder.Decode(&frame)
	if err != nil {
		return continuity.Frame{}, continuity.ErrState
	}

	var trailing any

	err = decoder.Decode(&trailing)
	if !errors.Is(err, io.EOF) {
		return continuity.Frame{}, continuity.ErrState
	}

	return frame, nil
}

func (file *File) replace(ctx context.Context, data []byte) error {
	temporary, err := os.CreateTemp(file.directory, ".state-*")
	if err != nil {
		return err
	}

	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporary.Name())
	}()

	_, err = temporary.Write(data)
	if err != nil {
		return err
	}

	err = temporary.Sync()
	if err != nil {
		return err
	}

	err = temporary.Close()
	if err != nil {
		return err
	}

	err = file.boundary(ctx, beforeRename)
	if err != nil {
		return err
	}

	err = os.Rename(temporary.Name(), filepath.Join(file.directory, "state.json"))
	if err != nil {
		return err
	}

	err = file.boundary(context.WithoutCancel(ctx), "afterRename")
	if err != nil {
		return err
	}

	directory, err := os.Open(file.directory)
	if err != nil {
		return err
	}

	defer func() { _ = directory.Close() }()

	return directory.Sync()
}

func (file *File) boundary(ctx context.Context, phase string) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	if file.hook != nil {
		return file.hook(phase)
	}

	return nil
}
