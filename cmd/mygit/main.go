package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

const (
	blob = "blob"
)

func catFile(w io.Writer, hash string) error {
	if len(hash) != 40 {
		return errors.New("invalid hash given")
	}

	path := path.Join(".git/objects", hash[:2], hash[2:])

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	zlr, err := zlib.NewReader(f)
	if err != nil {
		return fmt.Errorf("create zlib reader: %w", err)
	}
	defer zlr.Close()

	br := bufio.NewReader(zlr)
	for {
		b, err := br.ReadByte()
		if err != nil {
			return fmt.Errorf("read byte: %w", err)
		}

		if b == 0 {
			break
		}
	}

	io.Copy(w, br)
	return nil
}

func hashBlob(filePath string) (string, error) {
	origFile, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer origFile.Close()

	fi, err := origFile.Stat()
	if err != nil {
		return "", fmt.Errorf("get file info: %w", err)
	}

	// write header
	size := int(fi.Size())
	header := fmt.Sprintf("blob %d\u0000", size)

	var buf bytes.Buffer
	if _, err := buf.WriteString(header); err != nil {
		return "", fmt.Errorf("write header")
	}

	// write file's content after header
	buf.Grow(size)
	io.Copy(&buf, origFile)

	sha := sha1.New()
	sha.Write(buf.Bytes())
	hex := hex.EncodeToString(sha.Sum(nil))

	blobPath := path.Join(".git/objects", hex[:2])

	if err := os.MkdirAll(blobPath, 0770); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	blobPath = path.Join(blobPath, hex[2:])
	blobFile, err := os.Create(blobPath)
	if err != nil {
		return "", fmt.Errorf("create blob file: %w", err)
	}
	defer blobFile.Close()

	// write compressed contents (header+file contents)
	zlw := zlib.NewWriter(blobFile)
	if _, err := zlw.Write(buf.Bytes()); err != nil {
		return "", fmt.Errorf("write to blob file: %w", err)
	}

	return hex, nil
}

func ensureArgsLen(ln int) {
	if len(os.Args) < ln {
		fmt.Fprintf(os.Stderr, "Invalid number of arguments\n")
		os.Exit(1)
	}
}

// Usage: your_git.sh <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/master\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")

	case "cat-file":
		ensureArgsLen(4)

		if err := catFile(os.Stdout, os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to cat file: %v\n", err)
			os.Exit(1)
		}

	case "hash-object":
		ensureArgsLen(4)

		hash, err := hashBlob(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to hash object: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprint(os.Stdout, hash)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
