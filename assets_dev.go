//go:build !prod

package main

import (
	"io/fs"
)

func ExtractAll(dest string) error {
	return nil
}

func ExtractDist(dest string) error {
	return nil
}

func ExtractFfmpeg(dest string) error {
	return nil
}

func ExtractFfplay(dest string) error {
	return nil
}

func ExtractSetting(dest string) error {
	return nil
}

func ReadFile(path string) ([]byte, error) {
	return nil, nil
}

func Open(path string) (fs.File, error) {
	return nil, nil
}
