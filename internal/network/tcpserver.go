package network

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type TCPServer struct {
	address string // The address used to bind the listener to
	port    int    // The port used to bind the listener to

	done        chan struct{} // A channel used to signal the server to stop
	debug       bool          // Whether or not to print debug messages
	connections []*net.Conn   // A slice of connections
	packets     chan *Packet  // A channel used to send packets to the server
	connMutex   sync.Mutex    // A  mutex to sync access to connections
}

// NewTCPServer creates a new TCPServer instance
func NewTCPServer(address string, port int, debug bool) *TCPServer {
	return &TCPServer{
		address:     address,
		port:        port,
		done:        make(chan struct{}),
		debug:       debug,
		connections: make([]*net.Conn, 0),
		packets:     make(chan *Packet),
		connMutex:   sync.Mutex{},
	}
}

// Listen starts listening for incoming connections
func (s *TCPServer) Listen() error {
	// Create a listener
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.address, s.port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// The goroutine that will accept incoming connections
	go func() {
		for {
			select {
			case <-s.done:
				// The done channel has been closed, exit the goroutine
				return
			default:
				// Accept a connection
				c, err := l.Accept()
				if err != nil {
					// Log the error and continue accepting connections
					log.Println("Error accepting connection: ", err.Error())
				} else {
					// Handle the connection
					go s.handleConnection(c)
				}
			}
		}
	}()

	return nil
}

// handleConnection handles a connection, this is run as a goroutine and there will be one for each connection
func (s *TCPServer) handleConnection(c net.Conn) {
	s.connMutex.Lock()
	s.connections = append(s.connections, &c)
	s.connMutex.Unlock()

	defer func() {
		// Close the connection and remove it from the slice of connections
		s.connMutex.Lock()
		for i, conn := range s.connections {
			if conn == &c {
				s.connections = append(s.connections[:i], s.connections[i+1:]...)
				break
			}
		}
		s.connMutex.Unlock()
	}()

	if s.debug {
		log.Println("Handling connection from: ", c.RemoteAddr().String())
	}
	for {
		select {
		case <-s.done:
			// The done channel has been closed, exit the goroutine
			log.Println("Stopping handling connection from: ", s.address)
			return
		default:
			// Get the packet ID from data
			packetID := make([]byte, 1)
			_, err := c.Read(packetID)
			if err != nil {
				// Log Error then stop handling the connection
				log.Println("Error reading packet ID: ", err.Error())
				return
			}
			data := make([]byte, PacketSizes[packetID[0]])
			_, err = c.Read(data)
			if err != nil {
				// Log Error then stop handling the connection
				log.Println("Error reading packet data: ", err.Error())
				return
			}

			// Get the ID (index) of the connection
			var id uint32
			for i, conn := range s.connections {
				if conn == &c {
					id = uint32(i)
					break
				}
			}

			packet := Packet{
				data:     append(packetID, data...),
				size:     PacketSizes[packetID[0]],
				senderID: id,
			}
			s.packets <- &packet
		}
	}
}

func (s *TCPServer) Stop() {
	// Close the done channel, this will cause the goroutine in Listen() to exit, it will also stop accepting new connections
	close(s.done)
}
