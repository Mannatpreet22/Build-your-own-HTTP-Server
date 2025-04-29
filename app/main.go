package main

import (
	"bytes"
	"compress/gzip"
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

func ReadEncodingHeader(str string) string {
	arr := strings.Split(str, "\r\n")
	for _, line := range arr {
		if strings.HasPrefix(line, "Accept-Encoding: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Accept-Encoding: "))
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

func Gzip(encodingHeader string) bool {
	if strings.Contains(encodingHeader, "gzip") {
		return true
	}
	return false
}

func ReadConnectionHeader(str string) string {
	arr := strings.Split(str, "\r\n")
	for _, line := range arr {
		if strings.HasPrefix(line, "Connection: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Connection: "))
		}
	}
	return ""
}

func parseRequest(conn net.Conn, directory string) {
	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			return
		}
		message := string(buffer[:n])
		encodedHeader := ReadEncodingHeader(message)
		extractString := SplitAndReturn(message)
		userAgent := ReadHeaders(message)
		requestMethod := strings.Split(message, " ")[0]
		connectionHeader := ReadConnectionHeader(message)

		// Base response headers
		baseHeaders := "HTTP/1.1 200 OK\r\n"
		if connectionHeader == "close" {
			baseHeaders += "Connection: close\r\n"
		}

		if strings.HasPrefix(extractString, "/echo/") {
			echoContent := strings.TrimPrefix(extractString, "/echo/")
			var compressedContent []byte
			if Gzip(encodedHeader) {
				var b bytes.Buffer
				w := gzip.NewWriter(&b)
				w.Write([]byte(echoContent))
				w.Close()
				compressedContent = b.Bytes()
			} else {
				compressedContent = []byte(echoContent)
			}

			response := ""
			if Gzip(encodedHeader) {
				response = fmt.Sprintf("%sContent-Encoding: gzip\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", baseHeaders, len(compressedContent), compressedContent)
			} else {
				response = fmt.Sprintf("%sContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", baseHeaders, len(compressedContent), compressedContent)
			}
			conn.Write([]byte(response))
		} else if strings.HasPrefix(extractString, "/user-agent") {
			var compressedContent []byte
			if Gzip(encodedHeader) {
				var b bytes.Buffer
				w := gzip.NewWriter(&b)
				w.Write([]byte(userAgent))
				w.Close()
				compressedContent = b.Bytes()
			} else {
				compressedContent = []byte(userAgent)
			}

			response := ""
			if Gzip(encodedHeader) {
				response = fmt.Sprintf("%sContent-Encoding: gzip\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", baseHeaders, len(compressedContent), compressedContent)
			} else {
				response = fmt.Sprintf("%sContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", baseHeaders, len(compressedContent), compressedContent)
			}
			conn.Write([]byte(response))
		} else if strings.HasPrefix(extractString, "/files/") {
			filename := strings.TrimPrefix(extractString, "/files/")
			filepath := filepath.Join(directory, filename)

			if requestMethod == "GET" {
				content, err := os.ReadFile(filepath)
				if err != nil {
					response := "HTTP/1.1 404 Not Found\r\n"
					if connectionHeader == "close" {
						response += "Connection: close\r\n"
					}
					response += "\r\n"
					conn.Write([]byte(response))
					continue
				}

				var compressedContent []byte
				if Gzip(encodedHeader) {
					var b bytes.Buffer
					w := gzip.NewWriter(&b)
					w.Write(content)
					w.Close()
					compressedContent = b.Bytes()
				} else {
					compressedContent = content
				}

				response := ""
				if Gzip(encodedHeader) {
					response = fmt.Sprintf("%sContent-Encoding: gzip\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", baseHeaders, len(compressedContent), compressedContent)
				} else {
					response = fmt.Sprintf("%sContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", baseHeaders, len(compressedContent), compressedContent)
				}

				conn.Write([]byte(response))
			} else if requestMethod == "POST" {
				body := ExtractRequestBody(message)
				file, err := os.Create(filepath)
				if err != nil {
					response := "HTTP/1.1 500 Internal Server Error\r\n"
					if connectionHeader == "close" {
						response += "Connection: close\r\n"
					}
					response += "\r\n"
					conn.Write([]byte(response))
					continue
				}
				file.WriteString(body)
				file.Close()
				response := "HTTP/1.1 201 Created\r\n"
				if connectionHeader == "close" {
					response += "Connection: close\r\n"
				}
				response += "\r\n"
				conn.Write([]byte(response))
			}
		} else if strings.HasPrefix(string(buffer), "GET / HTTP/1.1") {
			response := baseHeaders + "\r\n"
			conn.Write([]byte(response))
		} else {
			response := "HTTP/1.1 404 Not Found\r\n"
			if connectionHeader == "close" {
				response += "Connection: close\r\n"
			}
			response += "\r\n"
			conn.Write([]byte(response))
		}

		// Check if client requested connection close
		if connectionHeader == "close" {
			conn.Close()
			return
		}
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Code execution starts here!")

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
