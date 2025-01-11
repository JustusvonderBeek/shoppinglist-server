package util

import (
	"os"
	"path/filepath"
)

func ReadFileFromRoot(filename string) ([]byte, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootPath := filepath.Join(pwd, filename)
	return os.ReadFile(rootPath)
}

func WriteFileAtRoot(filename string, content []byte) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	rootPath := filepath.Join(pwd, filename)
	err = os.WriteFile(rootPath, content, 0640)
	return rootPath, err
}
