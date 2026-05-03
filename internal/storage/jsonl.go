package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
		return err
	}
	return AppendRaw(path, line)
}

func AppendRaw(path string, line []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	lock := flock.New(path + ".lock")
	ctx, cancel := context.WithTimeout(context.Background(), flockTimeout)
	defer cancel()
	locked, err := lock.TryLockContext(ctx, 20*time.Millisecond)
	if err != nil {
		return err
	}
	if !locked {
		return errors.New("flock timeout")
	}
	defer func() { _ = lock.Unlock() }()

	//nolint:gosec // path は呼び出し側が組み立てる JSONL の絶対パス。XDG state 配下で書き込み先が variable になるのは設計上の意図。
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(line); err != nil {
		return err
	}
	if _, err := f.Write([]byte{'\n'}); err != nil {
		return err
	}
	return nil
}

func ReadJSONL(path string, fn func(raw []byte) error) error {
	//nolint:gosec // path は呼び出し側が組み立てる JSONL の絶対パス。XDG state 配下で読込先が variable になるのは設計上の意図。
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
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
			return err
		}
	}
	return scanner.Err()
}
