package main

import (
	"log"
	"net"
	"syscall"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Printf("tcp-failer: listening on :80, will RST on accept")
	for {
		c, err := ln.(*net.TCPListener).AcceptTCP()
		if err != nil {
			// brief sleep to avoid tight loop on transient errors
			time.Sleep(50 * time.Millisecond)
			continue
		}
		// Set SO_LINGER to 0 to cause RST on close
		rawConn, err := c.SyscallConn()
		if err == nil {
			_ = rawConn.Control(func(fd uintptr) {
				linger := &syscall.Linger{Onoff: 1, Linger: 0}
				_ = syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, linger)
			})
		}
		// Close immediately to emit RST to the peer
		_ = c.Close()
	}
}
