package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteJSONToFile[T any](filePath string, data T) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ошибка при создании каталога: %w", err)
	}

	file, err := os.CreateTemp(dir, ".chat-data-*.tmp")
	if err != nil {
		return fmt.Errorf("ошибка при создании временного файла: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	mode := os.FileMode(0o644)
	if existing, statErr := os.Stat(filePath); statErr == nil {
		mode = existing.Mode().Perm()
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return fmt.Errorf("ошибка настройки прав файла: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		file.Close()
		return fmt.Errorf("ошибка при записи данных в файл: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("ошибка синхронизации файла: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия файла: %w", err)
	}
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("ошибка атомарной замены файла: %w", err)
	}

	return nil
}

func ReadJSONFromFile[T any](filePath string, data *T) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка при открытии файла: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(data); err != nil {
		return fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return nil
}

func CheckFileExists(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}

	exists := fileInfo != nil

	return exists, err
}
