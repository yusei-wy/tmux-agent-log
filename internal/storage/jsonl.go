package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const flockTimeout = 500 * time.Millisecond

const readBufferSize = 1 << 20

func AppendJSONL(path string, v any) error {
	line, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal jsonl entry: %w", err)
	}
	return AppendRaw(path, line)
}

func AppendRaw(path string, line []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create dir for jsonl: %w", err)
	}

	lock := flock.New(path + ".lock")
	ctx, cancel := context.WithTimeout(context.Background(), flockTimeout)
	defer cancel()
	locked, err := lock.TryLockContext(ctx, 20*time.Millisecond)
	if err != nil {
		return fmt.Errorf("acquire flock %s: %w", path, err)
	}
	if !locked {
		return fmt.Errorf("acquire flock %s: timeout", path)
	}
	defer func() { _ = lock.Unlock() }()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open jsonl %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("write jsonl line: %w", err)
	}
	if _, err := f.Write([]byte{'\n'}); err != nil {
		return fmt.Errorf("write jsonl newline: %w", err)
	}
	return nil
}

func ReadJSONL(path string, fn func(raw []byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open jsonl %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), readBufferSize)
	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		if err := fn(raw); err != nil {
			return fmt.Errorf("process jsonl line in %s: %w", path, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan jsonl %s: %w", path, err)
	}
	return nil
}
