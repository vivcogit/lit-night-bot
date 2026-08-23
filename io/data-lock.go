package io

import (
	"errors"
	"fmt"
	"lit-night-bot/utils"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var ErrDataDirectoryLocked = errors.New("каталог данных уже используется другим процессом")

// DataDirectoryLock prevents multiple bot or migration processes from writing
// the same collection of JSON files concurrently.
type DataDirectoryLock struct {
	file *os.File
}

func (iocd *IoChatData) TryAcquireDataDirectoryLock() (*DataDirectoryLock, error) {
	lockDir := filepath.Join(iocd.dataPath, "_migration")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return nil, fmt.Errorf("не удалось создать каталог блокировки: %w", err)
	}
	if err := utils.SyncDirectory(iocd.dataPath); err != nil {
		return nil, fmt.Errorf("не удалось синхронизировать каталог блокировки: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(lockDir, "writer.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть блокировку: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, ErrDataDirectoryLocked
		}
		return nil, fmt.Errorf("не удалось захватить блокировку: %w", err)
	}
	return &DataDirectoryLock{file: file}, nil
}

func (lock *DataDirectoryLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	file := lock.file
	lock.file = nil
	if err := unix.Flock(int(file.Fd()), unix.LOCK_UN); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
