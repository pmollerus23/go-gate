package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestServeForwardsTrafficToBackend(t *testing.T) {
	backendListener := listenOnRandomPort(t)
	defer backendListener.Close()

	backendDone := make(chan error, 1)
	go func() {
		conn, err := backendListener.Accept()
		if err != nil {
			backendDone <- err
			return
		}
		defer conn.Close()

		_, err = io.Copy(conn, conn)
		backendDone <- err
	}()

	proxyListener := listenOnRandomPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	proxyDone := make(chan error, 1)
	go func() {
		proxyDone <- serve(ctx, proxyListener, backendListener.Addr().String())
	}()

	client, err := net.Dial("tcp", proxyListener.Addr().String())
	if err != nil {
		cancel()
		t.Fatalf("dial proxy: %v", err)
	}
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		client.Close()
		cancel()
		t.Fatalf("set client deadline: %v", err)
	}

	message := []byte("hello through the proxy")
	if _, err := client.Write(message); err != nil {
		client.Close()
		cancel()
		t.Fatalf("write to proxy: %v", err)
	}

	response := make([]byte, len(message))
	if _, err := io.ReadFull(client, response); err != nil {
		client.Close()
		cancel()
		t.Fatalf("read from proxy: %v", err)
	}
	if string(response) != string(message) {
		t.Errorf("response = %q, want %q", response, message)
	}

	client.Close()
	cancel()

	waitForResult(t, "proxy", proxyDone)
	waitForResult(t, "backend", backendDone)
}

func TestServeHandlesUnavailableBackend(t *testing.T) {
	backendListener := listenOnRandomPort(t)
	unavailableAddress := backendListener.Addr().String()
	if err := backendListener.Close(); err != nil {
		t.Fatalf("close backend listener: %v", err)
	}

	proxyListener := listenOnRandomPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	proxyDone := make(chan error, 1)
	go func() {
		proxyDone <- serve(ctx, proxyListener, unavailableAddress)
	}()

	client, err := net.Dial("tcp", proxyListener.Addr().String())
	if err != nil {
		cancel()
		t.Fatalf("dial proxy: %v", err)
	}
	defer client.Close()

	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		cancel()
		t.Fatalf("set client deadline: %v", err)
	}

	buffer := make([]byte, 1)
	_, err = client.Read(buffer)
	if err == nil {
		cancel()
		t.Fatal("expected proxy to close client connection after backend dial failed")
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		cancel()
		t.Fatal("proxy left client connected after backend dial failed")
	}

	select {
	case err := <-proxyDone:
		t.Fatalf("proxy stopped after one backend dial failure: %v", err)
	default:
	}

	cancel()
	waitForResult(t, "proxy", proxyDone)
}

func TestServeForwardsArbitraryBytes(t *testing.T) {
	backendListener := listenOnRandomPort(t)
	defer backendListener.Close()

	backendDone := make(chan error, 1)
	go func() {
		conn, err := backendListener.Accept()
		if err != nil {
			backendDone <- err
			return
		}
		defer conn.Close()

		_, err = io.Copy(conn, conn)
		backendDone <- err
	}()

	proxyListener := listenOnRandomPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	proxyDone := make(chan error, 1)
	go func() {
		proxyDone <- serve(ctx, proxyListener, backendListener.Addr().String())
	}()

	client, err := net.Dial("tcp", proxyListener.Addr().String())
	if err != nil {
		cancel()
		t.Fatalf("dial proxy: %v", err)
	}
	defer client.Close()
	defer cancel()

	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}

	message := bytes.Repeat(
		[]byte{0x00, 0xff, 0x01, '\n', '\r', 0x7f, 0x80},
		20_000,
	)

	if _, err := client.Write(message); err != nil {
		client.Close()
		cancel()
		t.Fatalf("write to proxy: %v", err)
	}

	response := make([]byte, len(message))
	if _, err := io.ReadFull(client, response); err != nil {
		client.Close()
		cancel()
		t.Fatalf("read from proxy: %v", err)
	}
	if !bytes.Equal(response, message) {
		t.Errorf(
			"response differs from message: got %d bytes, want %d",
			len(response),
			len(message),
		)
	}

	client.Close()
	cancel()

	waitForResult(t, "proxy", proxyDone)
	waitForResult(t, "backend", backendDone)

}

func TestServeHandlesConcurrentClients(t *testing.T) {
	backendListener := listenOnRandomPort(t)
	defer backendListener.Close()

	go func() {
		for {
			conn, err := backendListener.Accept()
			if err != nil {
				return
			}

			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()

	proxyListener := listenOnRandomPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	proxyDone := make(chan error, 1)
	go func() {
		proxyDone <- serve(ctx, proxyListener, backendListener.Addr().String())
	}()

	const clientCount = 10

	var clientWG sync.WaitGroup
	results := make(chan error, clientCount)

	for i := 0; i < clientCount; i++ {
		clientWG.Add(1)

		go func(clientID int) {
			defer clientWG.Done()

			client, err := net.Dial("tcp", proxyListener.Addr().String())
			if err != nil {
				results <- fmt.Errorf("client %d: dial proxy: %w", clientID, err)
				return
			}
			defer client.Close()

			if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
				results <- fmt.Errorf("client %d: set deadline: %w", clientID, err)
				return
			}

			message := []byte(fmt.Sprintf("message-from-client-%d\n", clientID))
			if _, err := client.Write(message); err != nil {
				results <- fmt.Errorf("client %d: write: %w", clientID, err)
				return
			}

			response := make([]byte, len(message))
			if _, err := io.ReadFull(client, response); err != nil {
				results <- fmt.Errorf("client %d: read: %w", clientID, err)
				return
			}

			if !bytes.Equal(response, message) {
				results <- fmt.Errorf("client %d: response %q, want %q", clientID, response, message)
				return
			}

			results <- nil
		}(i)
	}

	clientWG.Wait()
	close(results)

	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}

	cancel()
	waitForResult(t, "proxy", proxyDone)
}

func listenOnRandomPort(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return listener
}

func waitForResult(t *testing.T, name string, result <-chan error) {
	t.Helper()

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("%s stopped with an error: %v", name, err)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timed out waiting for %s to stop", name)
	}
}
