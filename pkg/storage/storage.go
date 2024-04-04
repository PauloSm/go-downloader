package storage

import (
	"os"
)

type LocalStorage struct{}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

func (l *LocalStorage) Save(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

func (l *LocalStorage) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *LocalStorage) Delete(path string) error {
	return os.Remove(path)
}
