package main

import (
"fmt"
"os"
"time"
)

func init() {
f, _ := os.Create("RCE_POC_was_here.txt")
defer f.Close()
fmt.Fprintln(f, "init() executed at", time.Now().Format(time.RFC3339))
fmt.Println("::RCE-POC:: init() executed")
}
