package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

func WriteJSONToFile[T any](filePath string, data T) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("ошибка при записи данных в файл: %w", err)
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
