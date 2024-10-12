package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("Server is running at ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "Expected WebSocket Upgrade", http.StatusBadRequest)
		return
	}

	conn, err := wsHandshake(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connection established")

	for {
		messageType, msg, err := wsReadMessage(conn)
		if err != nil {
			if err == io.EOF {
				log.Println("Client closed the connection")
			} else {
				log.Printf("Failed to read message: %v", err)
			}
			break
		}

		if messageType == 8 { // Close frame
			log.Println("Received close frame, closing connection")
			break
		}

		if messageType != 1 { // We only process text frames
			continue
		}

		go runCodeAndSendOutput(conn, string(msg))
	}
}

func runCodeAndSendOutput(conn *websocket, code string) {
	tmpDir, err := os.MkdirTemp("", "goplayground")
	if err != nil {
		sendError(conn, fmt.Sprintf("Failed to create temporary directory: %v", err))
		return
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		sendError(conn, fmt.Sprintf("Failed to write temporary file: %v", err))
		return
	}

	cmd := exec.Command("go", "run", tmpFile)
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(tmpDir, "go-cache"),
		"GOPATH="+filepath.Join(tmpDir, "go-path"),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendError(conn, fmt.Sprintf("Failed to create output pipe: %v", err))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		sendError(conn, fmt.Sprintf("Failed to create error output pipe: %v", err))
		return
	}

	if err := cmd.Start(); err != nil {
		sendError(conn, fmt.Sprintf("Failed to start command: %v", err))
		return
	}

	done := make(chan bool)
	go func() {
		reader := bufio.NewReader(io.MultiReader(stdout, stderr))
		var buffer strings.Builder
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading output: %v", err)
				}
				break
			}

			if r == '\x0c' { // Clear screen character
				if buffer.Len() > 0 {
					sendOutput(conn, buffer.String())
					buffer.Reset()
				}
				sendClearScreen(conn)
			} else {
				buffer.WriteRune(r)
				if r == '\n' || buffer.Len() >= 1024 { // Send complete line or when buffer reaches a certain size
					sendOutput(conn, buffer.String())
					buffer.Reset()
				}
			}
		}
		if buffer.Len() > 0 {
			sendOutput(conn, buffer.String())
		}
		done <- true
	}()

	go func() {
		time.Sleep(2 * time.Minute)
		if cmd.Process != nil {
			cmd.Process.Kill()
			sendError(conn, "Execution timeout, terminated")
		}
	}()

	<-done
	cmd.Wait()
}

func sendOutput(conn *websocket, output string) {
	msg := fmt.Sprintf(`{"type":"output","data":%q}`, output)
	err := wsSendMessage(conn, []byte(msg))
	if err != nil {
		log.Printf("Failed to send output: %v", err)
	}
}

func sendError(conn *websocket, errMsg string) {
	msg := fmt.Sprintf(`{"type":"error","data":%q}`, errMsg)
	err := wsSendMessage(conn, []byte(msg))
	if err != nil {
		log.Printf("Failed to send error: %v", err)
	}
}

func sendClearScreen(conn *websocket) {
	msg := `{"type":"clear"}`
	err := wsSendMessage(conn, []byte(msg))
	if err != nil {
		log.Printf("Failed to send clear screen command: %v", err)
	}
}

// WebSocket related functions

func wsHandshake(w http.ResponseWriter, r *http.Request) (*websocket, error) {
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, fmt.Errorf("Sec-WebSocket-Key is missing")
	}

	hash := sha1.New()
	hash.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("webserver doesn't support hijacking")
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	// Send WebSocket handshake response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	if _, err = bufrw.WriteString(response); err != nil {
		conn.Close()
		return nil, err
	}
	if err = bufrw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}

	return &websocket{Conn: conn}, nil
}

func wsReadMessage(conn *websocket) (messageType int, p []byte, err error) {
	var header [2]byte
	if _, err = io.ReadFull(conn.Conn, header[:]); err != nil {
		return
	}

	final := header[0]&0x80 != 0
	messageType = int(header[0] & 0x0f)
	masked := header[1]&0x80 != 0
	payloadLen := int(header[1] & 0x7f)

	if payloadLen == 126 {
		var extLen [2]byte
		if _, err = io.ReadFull(conn.Conn, extLen[:]); err != nil {
			return
		}
		payloadLen = int(extLen[0])<<8 | int(extLen[1])
	} else if payloadLen == 127 {
		var extLen [8]byte
		if _, err = io.ReadFull(conn.Conn, extLen[:]); err != nil {
			return
		}
		payloadLen = int(extLen[0])<<56 | int(extLen[1])<<48 | int(extLen[2])<<40 | int(extLen[3])<<32 |
			int(extLen[4])<<24 | int(extLen[5])<<16 | int(extLen[6])<<8 | int(extLen[7])
	}

	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(conn.Conn, maskKey[:]); err != nil {
			return
		}
	}

	p = make([]byte, payloadLen)
	if _, err = io.ReadFull(conn.Conn, p); err != nil {
		return
	}

	if masked {
		for i := range p {
			p[i] ^= maskKey[i%4]
		}
	}

	if !final {
		err = fmt.Errorf("non-final frames are not supported")
		return
	}

	return
}

func wsSendMessage(conn *websocket, message []byte) error {
	// Simplified WebSocket message sending implementation
	length := len(message)
	var header []byte

	if length <= 125 {
		header = []byte{0x81, byte(length)}
	} else if length <= 65535 {
		header = []byte{0x81, 126, byte(length >> 8), byte(length)}
	} else {
		header = []byte{0x81, 127}
		for i := 7; i >= 0; i-- {
			header = append(header, byte(length>>(8*i)))
		}
	}

	if _, err := conn.Conn.Write(header); err != nil {
		return err
	}

	if _, err := conn.Conn.Write(message); err != nil {
		return err
	}

	return nil
}

type websocket struct {
	Conn net.Conn
}

func (ws *websocket) Close() error {
	return ws.Conn.Close()
}
