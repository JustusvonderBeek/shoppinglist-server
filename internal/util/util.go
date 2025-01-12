package util

import (
	"os"
	"path/filepath"
)

type IReader interface {
	ReadConfig() ([]byte, error)
}

type ConfigReader struct {
	Filename string
}

func (c *ConfigReader) ReadConfig() ([]byte, error) {
	return ReadFileFromRoot(c.Filename)
}

func ReadFileFromRoot(filename string) ([]byte, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootPath := filepath.Join(pwd, filename)
	return os.ReadFile(rootPath)
}

func OverwriteFileAtRoot(content []byte, filename string) (string, int, error) {
	return WriteFileAtRoot(content, filename, true)
}

func WriteFileAtRoot(content []byte, filename string, overwrite bool) (string, int, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", 0, err
	}
	rootPath := filepath.Join(pwd, filename)
	fileMode := os.O_CREATE | os.O_WRONLY
	if overwrite {
		fileMode = fileMode | os.O_TRUNC
	} else {
		fileMode = fileMode | os.O_APPEND
	}
	file, err := os.OpenFile(rootPath, fileMode, 0660)
	if err != nil {
		return "", 0, err
	}
	n, err := file.Write(content)
	return rootPath, n, err
}
