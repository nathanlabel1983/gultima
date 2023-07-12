package network

import (
	"fmt"
	"log"
	"net"
)

type TCPServer struct {
	address string // The address used to bind the listener to
	port    int    // The port used to bind the listener to

	done    chan struct{} // A channel used to signal the server to stop
	packets chan *Packet  // A channel used to send packets to the server
}

// NewTCPServer creates a new TCPServer instance
func NewTCPServer(address string, port int) *TCPServer {
	return &TCPServer{
		address: address,
		port:    port,
		done:    make(chan struct{}),
		packets: make(chan *Packet),
	}
}

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
	log.Println("Handling connection from: ", c.RemoteAddr().String())

	select {
	case <-s.done:
		// The done channel has been closed, exit the goroutine
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
		packet := Packet{
			data:     append(packetID, data...),
			size:     PacketSizes[packetID[0]],
			senderID: 0, // TODO: Get the sender ID from the connection
		}
		s.packets <- &packet
	}
}

func (s *TCPServer) Stop() {
	// Close the done channel, this will cause the goroutine in Listen() to exit, it will also stop accepting new connections
	close(s.done)
}
