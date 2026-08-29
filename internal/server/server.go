package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

const backendAddress = "127.0.0.1:8081"

func Run(ctx context.Context) error {
	fmt.Println("Running server ...")
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		return fmt.Errorf("listen on :8080: %w", err)
	}
	defer ln.Close()

	return serve(ctx, ln, backendAddress)
}

func serve(ctx context.Context, ln net.Listener, backendAddress string) error {
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	log.Printf("Listening on %s", ln.Addr())

	var wg sync.WaitGroup

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				log.Println("server shutting down")
				break
			}

			log.Println("accept:", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := handleConnection(conn, ctx, backendAddress); err != nil {
				log.Printf("handle connection: %v", err)
			}
		}()
	}

	wg.Wait()
	log.Println("server stopped")

	return nil
}

func handleConnection(
	conn net.Conn,
	ctx context.Context,
	backendAddress string,
) error {
	defer conn.Close()

	backendConn, err := (&net.Dialer{}).DialContext(
		ctx,
		"tcp",
		backendAddress,
	)
	if err != nil {
		return fmt.Errorf("dial backend %s: %w", backendAddress, err)
	}
	defer backendConn.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
			backendConn.Close()
		case <-done:
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go proxy(&wg, backendConn, conn)
	go proxy(&wg, conn, backendConn)

	wg.Wait()
	if ctx.Err() == nil {
		log.Printf("closed proxy connection to %s", backendAddress)
	}

	return nil
}

func proxy(wg *sync.WaitGroup, dst, src net.Conn) {
	defer wg.Done()

	if _, err := io.Copy(dst, src); err != nil {
		log.Printf("proxy %s -> %s: %v", src.RemoteAddr(), dst.RemoteAddr(), err)
	}

	if tcpConn, ok := dst.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
	}
}
