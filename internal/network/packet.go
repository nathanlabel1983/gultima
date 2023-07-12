package network

type Packet struct {
	data     []byte // The raw packet data
	size     uint16 // The size of the packet
	senderID uint32 // The ID of the sender
}
