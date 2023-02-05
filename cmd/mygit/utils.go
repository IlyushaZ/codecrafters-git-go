package main

import (
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

func headerValid(expectedType, have string) bool {
	split := strings.Split(have, " ")
	if len(split) != 2 {
		return false
	}

	if split[0] != expectedType {
		return false
	}

	return true
}

// TODO: remove this
func ensureArgsLen(ln int) {
	if len(os.Args) < ln {
		fmt.Fprintf(os.Stderr, "Invalid number of arguments\n")
		os.Exit(1)
	}
}

func hash(data []byte) []byte {
	sha := sha1.New()
	sha.Write(data)
	return sha.Sum(nil)
}

func hashToString(h []byte) string {
	return hex.EncodeToString(h)
}

func saveCompressed(hash []byte, data []byte) error {
	hex := hex.EncodeToString(hash)
	objPath := path.Join(".git/objects", hex[:2])

	if err := os.MkdirAll(objPath, 0770); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	objPath = path.Join(objPath, hex[2:])
	objFile, err := os.Create(objPath)
	if err != nil {
		return fmt.Errorf("create blob file: %w", err)
	}
	defer objFile.Close()

	// write compressed contents (header+file contents)
	zlw := zlib.NewWriter(objFile)
	if _, err := zlw.Write(data); err != nil {
		return fmt.Errorf("write to blob file: %w", err)
	}
	defer zlw.Close()

	return nil
}

func getDecompressedObject(hash string) (rc io.Reader, closeFn func(), err error) {
	path := path.Join(".git/objects", hash[:2], hash[2:])

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}

	zlr, err := zlib.NewReader(f)
	if err != nil {
		return nil, nil, fmt.Errorf("create zlib reader: %w", err)
	}

	return zlr, func() {
		f.Close()
		zlr.Close()
	}, nil
}
