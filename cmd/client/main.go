package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func run() error {
	fmt.Println("start client.")
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	defer conn.Close()

	serverReader := bufio.NewReader(conn)
	inputScanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Connected. Type a message and press Enter.")
	for inputScanner.Scan() {
		if _, err := fmt.Fprintln(conn, inputScanner.Text()); err != nil {
			return fmt.Errorf("send message: %w", err)
		}

		response, err := serverReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		fmt.Print(response)
	}

	if err := inputScanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
