package transport

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type WebSocketSSHTunnel struct {
	sshTransport *SSHTransport
	upgrader     websocket.Upgrader
	connections  map[*websocket.Conn]*SSHTunnelConnection
	connMutex    sync.RWMutex
}

type SSHTunnelConnection struct {
	WebSocketConn *websocket.Conn
	SSHConn       *SSHConnection
	SSHSession    *ssh.Session
	LocalAddr     string
	RemoteAddr    string
	CreatedAt     time.Time
	LastActivity  time.Time
	Active        bool
}

func NewWebSocketSSHTunnel(sshTransport *SSHTransport) *WebSocketSSHTunnel {
	return &WebSocketSSHTunnel{
		sshTransport: sshTransport,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
		connections: make(map[*websocket.Conn]*SSHTunnelConnection),
	}
}

func (wst *WebSocketSSHTunnel) HandleWebSocketOverSSH(w http.ResponseWriter, r *http.Request) error {
	// Upgrade HTTP connection to WebSocket
	wsConn, err := wst.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade to WebSocket: %w", err)
	}
	defer wsConn.Close()

	// Create SSH connection
	sshConn, err := wst.sshTransport.CreateSSHConnection(context.Background())
	if err != nil {
		wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "SSH connection failed"))
		return fmt.Errorf("failed to create SSH connection: %w", err)
	}
	defer wst.sshTransport.CloseSSHConnection(sshConn)

	// Create tunnel connection
	tunnelConn := &SSHTunnelConnection{
		WebSocketConn: wsConn,
		SSHConn:       sshConn,
		SSHSession:    sshConn.Session,
		LocalAddr:     sshConn.LocalAddr,
		RemoteAddr:    sshConn.RemoteAddr,
		CreatedAt:     time.Now(),
		LastActivity:  time.Now(),
		Active:        true,
	}

	// Register connection
	wst.connMutex.Lock()
	wst.connections[wsConn] = tunnelConn
	wst.connMutex.Unlock()

	// Cleanup on disconnect
	defer func() {
		wst.connMutex.Lock()
		delete(wst.connections, wsConn)
		wst.connMutex.Unlock()
		tunnelConn.Active = false
	}()

	log.Printf("WebSocket over SSH tunnel established: %s -> %s", tunnelConn.LocalAddr, tunnelConn.RemoteAddr)

	// Start bidirectional data forwarding
	return wst.forwardData(tunnelConn)
}

func (wst *WebSocketSSHTunnel) forwardData(conn *SSHTunnelConnection) error {
	// Create channels for bidirectional communication
	wsToSSH := make(chan []byte, 100)
	sshToWS := make(chan []byte, 100)
	errChan := make(chan error, 2)

	// Start WebSocket to SSH forwarding
	go func() {
		defer close(wsToSSH)
		for {
			_, message, err := conn.WebSocketConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				errChan <- err
				return
			}
			conn.LastActivity = time.Now()
			select {
			case wsToSSH <- message:
			case <-time.After(5 * time.Second):
				log.Printf("WebSocket to SSH channel full, dropping message")
			}
		}
	}()

	// Start SSH to WebSocket forwarding
	go func() {
		defer close(sshToWS)

		// Set up SSH session for data forwarding
		stdin, err := conn.SSHSession.StdinPipe()
		if err != nil {
			errChan <- fmt.Errorf("failed to get SSH stdin: %w", err)
			return
		}
		defer stdin.Close()

		stdout, err := conn.SSHSession.StdoutPipe()
		if err != nil {
			errChan <- fmt.Errorf("failed to get SSH stdout: %w", err)
			return
		}

		stderr, err := conn.SSHSession.StderrPipe()
		if err != nil {
			errChan <- fmt.Errorf("failed to get SSH stderr: %w", err)
			return
		}

		// Start SSH session
		if err := conn.SSHSession.Shell(); err != nil {
			errChan <- fmt.Errorf("failed to start SSH shell: %w", err)
			return
		}

		// Forward SSH stdout to WebSocket
		go func() {
			buffer := make([]byte, 4096)
			for {
				n, err := stdout.Read(buffer)
				if err != nil {
					if err != io.EOF {
						log.Printf("SSH stdout read error: %v", err)
					}
					return
				}
				conn.LastActivity = time.Now()
				select {
				case sshToWS <- buffer[:n]:
				case <-time.After(5 * time.Second):
					log.Printf("SSH to WebSocket channel full, dropping data")
				}
			}
		}()

		// Forward SSH stderr to WebSocket (as error messages)
		go func() {
			buffer := make([]byte, 4096)
			for {
				n, err := stderr.Read(buffer)
				if err != nil {
					if err != io.EOF {
						log.Printf("SSH stderr read error: %v", err)
					}
					return
				}
				conn.LastActivity = time.Now()
				// Send stderr as error message
				errorMsg := fmt.Sprintf("SSH Error: %s", string(buffer[:n]))
				select {
				case sshToWS <- []byte(errorMsg):
				case <-time.After(5 * time.Second):
					log.Printf("SSH stderr to WebSocket channel full, dropping error")
				}
			}
		}()

		// Forward WebSocket messages to SSH stdin
		for message := range wsToSSH {
			if _, err := stdin.Write(message); err != nil {
				log.Printf("Failed to write to SSH stdin: %v", err)
				break
			}
		}
	}()

	// Forward SSH output to WebSocket
	go func() {
		for message := range sshToWS {
			if err := conn.WebSocketConn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to write to WebSocket: %v", err)
				return
			}
			conn.LastActivity = time.Now()
		}
	}()

	// Wait for error or connection close
	select {
	case err := <-errChan:
		if err != nil && err != io.EOF {
			log.Printf("WebSocket over SSH tunnel error: %v", err)
		}
		return err
	case <-time.After(24 * time.Hour): // 24 hour timeout
		log.Println("WebSocket over SSH tunnel timeout")
		return fmt.Errorf("tunnel timeout")
	}
}

func (wst *WebSocketSSHTunnel) GetConnectionStats() map[string]interface{} {
	wst.connMutex.RLock()
	defer wst.connMutex.RUnlock()

	stats := map[string]interface{}{
		"total_connections":  len(wst.connections),
		"active_connections": 0,
		"connections":        make([]map[string]interface{}, 0),
	}

	activeCount := 0
	for _, conn := range wst.connections {
		if conn.Active {
			activeCount++
		}

		connInfo := map[string]interface{}{
			"local_addr":    conn.LocalAddr,
			"remote_addr":   conn.RemoteAddr,
			"created_at":    conn.CreatedAt,
			"last_activity": conn.LastActivity,
			"active":        conn.Active,
		}

		stats["connections"] = append(stats["connections"].([]map[string]interface{}), connInfo)
	}

	stats["active_connections"] = activeCount
	return stats
}

func (wst *WebSocketSSHTunnel) CloseAllConnections() {
	wst.connMutex.Lock()
	defer wst.connMutex.Unlock()

	for wsConn, tunnelConn := range wst.connections {
		tunnelConn.Active = false

		// Close WebSocket connection
		wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutdown"))
		wsConn.Close()

		// Close SSH connection
		if tunnelConn.SSHSession != nil {
			tunnelConn.SSHSession.Close()
		}
		if tunnelConn.SSHConn != nil {
			wst.sshTransport.CloseSSHConnection(tunnelConn.SSHConn)
		}
	}

	wst.connections = make(map[*websocket.Conn]*SSHTunnelConnection)
}

// SSH Tunnel with Port Forwarding
func (wst *WebSocketSSHTunnel) CreateSSHTunnel(localPort int, remoteHost string, remotePort int) error {
	sshConn, err := wst.sshTransport.CreateSSHConnection(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create SSH connection for tunnel: %w", err)
	}

	// Create local listener
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", localPort))
	if err != nil {
		wst.sshTransport.CloseSSHConnection(sshConn)
		return fmt.Errorf("failed to create local listener: %w", err)
	}

	// Start accepting connections and forwarding them through SSH
	go func() {
		defer listener.Close()
		defer wst.sshTransport.CloseSSHConnection(sshConn)

		for {
			localConn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept local connection: %v", err)
				break
			}

			// Forward connection through SSH tunnel
			go wst.forwardConnectionThroughSSH(sshConn, localConn, remoteHost, remotePort)
		}
	}()

	log.Printf("SSH tunnel established: localhost:%d -> %s:%d", localPort, remoteHost, remotePort)
	return nil
}

func (wst *WebSocketSSHTunnel) forwardConnectionThroughSSH(sshConn *SSHConnection, localConn net.Conn, remoteHost string, remotePort int) {
	defer localConn.Close()

	// Create SSH session for this connection
	session, err := sshConn.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create SSH session for tunnel: %v", err)
		return
	}
	defer session.Close()

	// Set up port forwarding through SSH
	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)

	// Use SSH port forwarding
	conn, err := sshConn.Client.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("Failed to dial remote address through SSH: %v", err)
		return
	}
	defer conn.Close()

	// Start bidirectional forwarding
	done := make(chan struct{}, 2)

	// Forward local to remote
	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(conn, localConn)
	}()

	// Forward remote to local
	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(localConn, conn)
	}()

	// Wait for either direction to complete
	<-done
}
