package main

import (
	"fmt"
	"os"
)

func main() {
	f, _ := os.Open("test")
	s, _ := f.Stat()

	fmt.Println(s.Size())
}
