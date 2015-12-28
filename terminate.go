package main

import (
	"os"
	"os/signal"
	"syscall"
)

// MakeTerminateChannel returns a channel that will become readable if
// the process is interrupted or terminated via a signal.
//
// This is used to gracefully exit the reader loop, which in turn causes
// the rest of the program to gracefully terminate, flushing any remaining
// buffers and writing its persistent state to disk.
func MakeTerminateChannel() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
