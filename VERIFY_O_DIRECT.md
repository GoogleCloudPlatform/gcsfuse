# Verifying O_DIRECT Flag Behavior

This document explains how to verify the behavior of the `O_DIRECT` flag in Linux, specifically in relation to the page cache. The `O_DIRECT` flag allows a program to bypass the page cache and read directly from the storage device. This can be useful in certain high-performance applications, but it's important to verify that it is behaving as expected.

## Page Cache

The page cache is a mechanism used by the Linux kernel to cache file data in memory. When a file is read, the data is stored in the page cache. Subsequent reads of the same file can be served from the page cache, which is much faster than reading from the storage device.

## O_DIRECT Flag

The `O_DIRECT` flag is a flag that can be passed to the `open()` system call. When this flag is used, file I_O is performed directly to and from the user-space buffer, bypassing the page cache. This can improve performance in some cases, but it can also have a negative impact if not used carefully.

## Verifying Cached Data (O_DIRECT unset)

When a file is read without the `O_DIRECT` flag, its data should be present in the page cache. Here's how to verify this:

1. **Create a test file:**
   ```bash
   dd if=/dev/urandom of=test_file bs=1M count=10
   ```

2. **Clear the page cache:**
   ```bash
   sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'
   ```

3. **Read the file using a simple Go program:**
   Create a file named `read_file.go`:
   ```go
   package main

   import (
	"io/ioutil"
	"log"
	"os"
   )

   func main() {
	file, err := os.Open("test_file")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
   }
   ```
   Run the program:
   ```bash
   go run read_file.go
   ```

4. **Check if the file is in the page cache:**
   You can use the `fincore` utility to check if a file is in the page cache.
   ```bash
   fincore --pages=false --summarize --only-cached test_file
   ```
   The output should show that a significant portion of the file is cached.

## Verifying Non-Cached Data (O_DIRECT set)

When a file is read with the `O_DIRECT` flag, its data should not be present in the page cache. Here's how to verify this:

1. **Create a test file:**
   ```bash
   dd if=/dev/urandom of=test_file_direct bs=1M count=10
   ```

2. **Clear the page cache:**
   ```bash
   sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'
   ```

3. **Read the file using `dd` with the `O_DIRECT` flag:**
   ```bash
   dd if=test_file_direct of=/dev/null bs=1M iflag=direct
   ```

4. **Check if the file is in the page cache:**
   ```bash
   fincore --pages=false --summarize --only-cached test_file_direct
   ```
   The output should show that the file is not cached.

## Conclusion

By using the techniques described in this document, you can verify the behavior of the `O_DIRECT` flag and ensure that it is being used correctly in your application. The `fincore` utility is a valuable tool for inspecting the page cache and understanding how your application is interacting with the file system.
