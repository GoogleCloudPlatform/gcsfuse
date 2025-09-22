package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// The hardcoded bash script to be executed before file operations.
const bashScript = `
#!/bin/bash
echo "============================================="
echo "Starting the mounting operation..."
echo "" > $PWD/details/gcsfuse.log
fusermount -uz "$PWD/details/bkt" || true

go run ./ --log-severity=trace --rename-dir-limit=1000000 --enable-google-lib-auth=false --write-global-max-blocks=0 --log-file=$PWD/details/gcsfuse.log mohitkyadav-hns-bkt-2 $PWD/details/bkt
rm -rf "$PWD/details/bkt/"*
echo "Hello" > $PWD/details/bkt/existing.txt
echo "" > $PWD/details/bkt/zero-bytes.txt
echo "" > $PWD/details/gcsfuse.log
fusermount -uz "$PWD/details/bkt" || true
go run ./ --log-severity=trace --rename-dir-limit=1000000 --enable-google-lib-auth=false --write-global-max-blocks=0 --log-file=$PWD/details/gcsfuse.log mohitkyadav-hns-bkt-2 $PWD/details/bkt
echo "============================================="
`

// A pre-seeded random number generator for local use.
// We seed it with the current time to ensure different results on each run.
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// letterCharset is the set of characters to choose from for the random string.
const letterCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString creates a random string of a given length.
func GenerateRandomString(length int) string {
	// Create a byte slice of the desired length.
	b := make([]byte, length)

	// Loop over the byte slice and fill it with random characters.
	for i := range b {
		// seededRand.Intn() returns a random integer in [0, n).
		// We use it to pick a random index from our character set.
		b[i] = letterCharset[seededRand.Intn(len(letterCharset))]
	}

	// Convert the byte slice to a string and return it.
	return string(b)
}

// The fixed size of data to write into each file (e.g., 10MB).
const dataSize = 1024 * 1024 * 10

func OpenAnExistingFileWithOTruncOption(dir string) {
	filePath := filepath.Join(dir, "existing.txt")

	fmt.Printf("Opening and truncating file: %s\n", filePath)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filePath, err)
	}
	defer f.Close()
}

func OpenAnExistingFileOfZeroByteOption(dir string) {
	filePath := filepath.Join(dir, "zero-byte.txt")

	fmt.Printf("Opening file: %s\n", filePath)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filePath, err)
	}
	f.WriteAt(make([]byte, dataSize), 0)
	defer f.Close()
}

func SequentialWriteThenOutOfOrderWrite(dir string) {
	filePath := filepath.Join(dir, GenerateRandomString(10))

	fmt.Printf("Opening and truncating file: %s\n", filePath)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filePath, err)
	}
	f.WriteAt(make([]byte, dataSize), 0)
	f.WriteAt(make([]byte, dataSize), dataSize)
	f.WriteAt(make([]byte, dataSize), dataSize)

	defer f.Close()
}

func SequentialWriteThenTruncatingDownwardsThenWrite(dir string) {
	filePath := filepath.Join(dir, GenerateRandomString(10))

	fmt.Printf("Opening and truncating file: %s\n", filePath)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filePath, err)
	}
	f.WriteAt(make([]byte, dataSize), 0)
	f.WriteAt(make([]byte, dataSize), dataSize)

	f.Truncate(0)
	f.WriteAt(make([]byte, dataSize), 0)
	f.WriteAt(make([]byte, dataSize), dataSize)

	defer f.Close()
}

func OpenAnExistingFileWithOTruncThenWriteSequentially(dir string) { // Have global max blocks OK, Don't have global max blocks OK
	filePath := filepath.Join(dir, "existing.txt")

	fmt.Printf("Opening and truncating file: %s\n", filePath)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filePath, err)
	}
	f.WriteAt(make([]byte, dataSize), 0)
	f.WriteAt(make([]byte, dataSize), dataSize)
	f.WriteAt(make([]byte, dataSize), dataSize)
	defer f.Close()
}

func main() {
	// Define and parse command-line flags.
	dir := flag.String("dir", "/home/mohitkyadav_google_com/improve-streaming-error-messages/gcsfuse/details/bkt/", "The directory to create files in.")
	fmt.Println("\n--- Running Bash Script ---")
	cmd := exec.Command("bash", "-c", bashScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Bash script execution failed: %v", err)
	}
	fmt.Println("--- Bash Script Finished ---")
	// 0 ✅    -1 ✅
	//OpenAnExistingFileWithOTruncOption(*dir)

	// 0 ✅    -1 ✅
	//SequentialWriteThenOutOfOrderWrite(*dir)

	// 0 ✅    -1 ✅
	//OpenAnExistingFileWithOTruncThenWriteSequentially(*dir)

	// 0 ✅    -1 ✅
	//SequentialWriteThenTruncatingDownwardsThenWrite(*dir)

	// 0 ✅    -1 ✅
	OpenAnExistingFileOfZeroByteOption(*dir)
}
