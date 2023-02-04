package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

var ErrInvalidHash = errors.New("hash is not sha-1")

func catFile(w io.Writer, hash string) error {
	if len(hash) != 40 {
		return ErrInvalidHash
	}

	decompressed, close, err := getDecompressedObject(hash)
	if err != nil {
		return err
	}
	defer close()

	br := bufio.NewReader(decompressed)
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

func hashBlob(filePath string) ([]byte, error) {
	origFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer origFile.Close()

	fi, err := origFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("get file info: %w", err)
	}

	// write header
	size := int(fi.Size())
	header := fmt.Sprintf("blob %d\u0000", size)

	var buf bytes.Buffer
	if _, err := buf.WriteString(header); err != nil {
		return nil, fmt.Errorf("write header")
	}

	// write file's content after header
	buf.Grow(size)
	io.Copy(&buf, origFile)

	hex := hash(buf.Bytes())
	if err := saveCompressed(hex, buf.Bytes()); err != nil {
		return nil, err
	}

	return hex, nil
}

func writeTree(root string) ([]byte, error) {
	dir, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var buf bytes.Buffer

	for _, entry := range dir {
		if entry.Name() == ".git" {
			continue
		}

		if entry.IsDir() {
			hash, err := writeTree(path.Join(root, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("write subtree: %w", err)
			}

			fmt.Fprintf(&buf, "40000 %s\u0000%s", entry.Name(), hash)
			continue
		}

		hash, err := hashBlob(path.Join(root, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("hash blob: %w", err)
		}

		fmt.Fprintf(&buf, "100644 %s\u0000%s", entry.Name(), hash)
	}

	var finalBuf bytes.Buffer
	finalBuf.Grow(buf.Len())

	fmt.Fprintf(&finalBuf, "tree %d\u0000", buf.Len())
	io.Copy(&finalBuf, &buf)

	hex := hash(finalBuf.Bytes())

	if err := saveCompressed(hex, finalBuf.Bytes()); err != nil {
		return nil, err
	}

	return hex, nil
}

func lsTree(w io.Writer, hash string) error {
	if len(hash) != 40 {
		return ErrInvalidHash
	}

	decompressed, close, err := getDecompressedObject(hash)
	if err != nil {
		return err
	}
	defer close()

	br := bufio.NewReader(decompressed)

	// read header
	header, err := br.ReadString('\x00')
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	if !headerValid("tree", header) {
		return fmt.Errorf("invalid header: %s", err)
	}

	for {
		entry, err := br.ReadString('\x00')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("read entry: %w", err)
		}

		entry = strings.TrimRight(entry, "\x00")
		split := strings.Split(entry, " ")

		fmt.Fprintln(w, split[1])

		br.Discard(20)
	}

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

		fmt.Print(hashToString(hash))

	case "ls-tree":
		ensureArgsLen(4)

		err := lsTree(os.Stdout, os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to ls tree: %v\n", err)
			os.Exit(1)
		}

	case "write-tree":
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get working dir: %v\n", err)
			os.Exit(1)
		}

		hash, err := writeTree(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write tree: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(hashToString(hash))

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
