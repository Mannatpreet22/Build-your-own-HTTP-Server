package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func SplitAndReturn(str string) string {
	str = strings.Split(str, " ")[1]
	str = strings.TrimSpace(str)
	return str
}

func ReadHeaders(str string) string {
	arr := strings.Split(str, "\r\n")
	for _, line := range arr {
		if strings.HasPrefix(line, "User-Agent: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "User-Agent: "))
		}
	}
	return ""
}

func ReadContentHeader(str string) string {
	arr := strings.Split(str, "\r\n")
	for _, line := range arr {
		if strings.HasPrefix(line, "Content-Type: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Content-Type: "))
		}
	}
	return ""
}

func ExtractRequestBody(str string) string {
	arr := strings.Split(str, "\r\n\r\n")
	return arr[1]
}

func parseRequest(conn net.Conn, directory string) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		os.Exit(1)
	}
	message := string(buffer[:n])
	fmt.Println("Buffer is the following: ", message)

	extractString := SplitAndReturn(message)
	userAgent := ReadHeaders(message)
	requestMethod := strings.Split(message, " ")[0]

	if strings.HasPrefix(extractString, "/echo/") {
		echoContent := strings.TrimPrefix(extractString, "/echo/")
		response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoContent), echoContent)
		conn.Write([]byte(response))
		return
	}

	if strings.HasPrefix(extractString, "/user-agent") {
		response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
		conn.Write([]byte(response))
		return
	}

	if strings.HasPrefix(extractString, "/files/") {
		filename := strings.TrimPrefix(extractString, "/files/")
		filepath := filepath.Join(directory, filename)

		if requestMethod == "GET" {
			content, err := os.ReadFile(filepath)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				return
			}

			response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(content), content)
			conn.Write([]byte(response))
			return
		} else if requestMethod == "POST" {
			body := ExtractRequestBody(message)
			file, err := os.Create(filepath)
			if err != nil {
				conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
				return
			}
			file.WriteString(body)
			file.Close()
			conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
			return
		}
	}

	res := strings.HasPrefix(string(buffer), "GET / HTTP/1.1")
	if !res {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}

	conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Code execution starts here!")

	// Get directory from command line args
	var directory string
	if len(os.Args) > 2 && os.Args[1] == "--directory" {
		directory = os.Args[2]
	}

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go parseRequest(conn, directory)
	}
}
