package main

import (
	"bufio"
	"compress/zlib"
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
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Invalid number of arguments\n")
			os.Exit(1)
		}

		if err := catFile(os.Stdout, os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to cat file: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
