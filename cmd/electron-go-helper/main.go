package main

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"

	"github.com/gibavargas/electron-go/internal/native"
)

func main() {
	goruntime.LockOSThread()
	result, handled, err := native.ExecuteCEFSubprocess(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "electron-go-helper: %v\n", err)
		os.Exit(1)
	}
	if !handled {
		fmt.Fprintln(os.Stderr, "electron-go-helper: no CEF subprocess arguments")
		os.Exit(1)
	}
	os.Exit(result.ExitCode)
}
