package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func RunClient() {
	fmt.Println("start client.")
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	serverReader := bufio.NewReader(conn)
	inputScanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Connected. Type a message and press Enter.")
	for inputScanner.Scan() {
		if _, err := fmt.Fprintln(conn, inputScanner.Text()); err != nil {
			log.Println("error sending:", err)
			return
		}

		response, err := serverReader.ReadString('\n')
		if err != nil {
			log.Println("error reading:", err)
			return
		}

		fmt.Print(response)
	}

	if err := inputScanner.Err(); err != nil {
		log.Println("error reading input:", err)
	}
}

func main() {
	RunClient()
}
