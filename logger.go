// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.
// borrowed from Gin library

package supernova

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	reset   = string([]byte{27, 91, 48, 109})
)

func getDebugMethod(r *Request) func() {
	// Start timer
	start := time.Now()
	return func() {
		path := r.URI().Path()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		clientIP := r.RemoteIP().String()
		method := r.GetMethod()
		statusCode := r.Response.StatusCode()
		var statusColor, methodColor string
		if isTerminal(os.Stdin.Fd()) {
			statusColor = colorForStatus(statusCode)
			methodColor = colorForMethod(method)
		}

		fmt.Printf("[Supernova] %v |%s %3d %s| %13v | %s |%s  %s %-7s %s\n",
			end.Format("2006/01/02 - 15:04:05"),
			statusColor, statusCode, reset,
			latency,
			clientIP,
			methodColor, reset, method,
			path,
		)
	}
}

// IsTerminal returns true if the given file descriptor is a terminal.
func isTerminal(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, 0x5401, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func colorForStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return green
	case code >= 300 && code < 400:
		return white
	case code >= 400 && code < 500:
		return yellow
	default:
		return red
	}
}

func colorForMethod(method string) string {
	switch method {
	case "GET":
		return blue
	case "POST":
		return cyan
	case "PUT":
		return yellow
	case "DELETE":
		return red
	case "PATCH":
		return green
	case "HEAD":
		return magenta
	case "OPTIONS":
		return white
	default:
		return reset
	}
}
