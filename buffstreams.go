package buffstreams

import ()

import (
	"encoding/binary"
	"errors"
	"github.com/Sirupsen/logrus"
	"net"
	"sync"
)

type BuffManager struct {
	dialedConnections map[string]net.Conn
	listeningSockets  map[string]net.Listener
	sync.RWMutex
}

func New() *BuffManager {
	bm := &BuffManager{
		dialedConnections: make(map[string]net.Conn),
		listeningSockets:  make(map[string]net.Listener),
	}
	return bm
}

type ListenCallback func([]byte) error

func formatAddress(address string, port string) string {
	return address + ":" + port
}

func (bm *BuffManager) StartListening(port string, cb ListenCallback) error {
	address := formatAddress("", port)
	receiveSocket, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	bm.startListening(address, receiveSocket, cb)
	return nil
}

func (bm *BuffManager) startListening(address string, socket net.Listener, cb ListenCallback) {
	bm.Lock()
	bm.listeningSockets[address] = socket
	bm.Unlock()

	go func(address string, listener net.Listener) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// alert error
				// Hmmm but how - I'm in goroutine land. Need a class level logger? Don't want to
				// tie people to the bleh default logger, also personally would prefer logrus for levels
			} else {
				// Hand this off and immediately listen for more
				go handleListenedConn(address, conn, cb)
			}
		}
	}(address, socket)
}

func handleListenedConn(address string, conn net.Conn, cb ListenCallback) {
	for {
		// Handle getting the data header
		logrus.Info("Reading")
		headerByteSize := MessageSizeToBitLength(4098) // MAKE THIS CONFIGURABLE once you have any idea how you're going to do that
		headerBuffer := make([]byte, headerByteSize)

		// First, read the number of bytes required to determine the message length
		_, err := readFromConnection(conn, headerBuffer)
		if err != nil && err.Error() == "EOF" {
			// Log the error we got from the call to read
			logrus.Error("Error reading from the connection. Likely it closed on the clients end")
			logrus.Error(err)
			conn.Close()
			return
		}

		// Now turn that buffer of bytes into an integer - represnts size of message body
		msgLength, bytesParsed := binary.Uvarint(headerBuffer)
		// Not sure what the correct way to handle these errors are. For now, bomb out
		if bytesParsed == 0 {
			// "Buffer too small"
			logrus.Error("0 Bytes parsed from header")
			logrus.Error(err)
			return
		} else if bytesParsed < 0 {
			// "Buffer overflow"
			logrus.Error("Less than zero bytes parsed from header?")
			logrus.Error(err)
			return
		}
		dataBuffer := make([]byte, msgLength)
		bytesLen, err := readFromConnection(conn, dataBuffer)
		if err != nil && err.Error() == "EOF" {
			// log the error from the call to read
			logrus.Error("Failure to read from connection")
			logrus.Error(err)
			conn.Close()
			return
		}

		// If we read bytes, there wasn't an error, or if there was it was only EOF
		// And readbytes + EOF is normal, just as readbytes + no err, next read 0 bytes EOF
		// So... we take action on the actual message data
		if bytesLen > 0 && (err == nil || (err != nil && err.Error() == "EOF")) {
			// I ultimately have some design choices here
			// Currently, I am invoking a delegate thats been passed down the stack
			// I could...
			// Keep it as is
			// Tie the protobuffs library deeper into it, and take a reference to
			// the type that the message we're serializing is, and do that work too
			// But you'd still need a callback, unless...
			// I just push it onto a queue (not a slow ass channel, but a queue)
			// which has a reference passed down to it, and the main process
			// spawns a goroutine to reap off the queue and handle those in parallel

			// Callback, atm
			logrus.Info("Callback")
			cb(dataBuffer)
		}
	}
}

func readFromConnection(reader net.Conn, buffer []byte) (int, error) {
	// This fills the buffer
	bytesLen, err := reader.Read(buffer)
	// Output the content of the bytes to the queue
	if bytesLen == 0 {
		if err != nil && err.Error() == "EOF" {
			// "End of individual transmission"
			// We're just done reading from that conn
			return bytesLen, err
		}
	}

	if err != nil {
		//"Underlying network failure?"
		// Not sure what this error would be, but it could exist and i've seen it handled
		// as a general case in other networking code. Following in the footsteps of (greatness|madness)
	}
	// Read some bytes, return the length
	return bytesLen, nil
}

func (bm *BuffManager) dialOut(ip string, port string) error {
	address := formatAddress(ip, port)
	if _, ok := bm.dialedConnections[address]; ok == true {
		// Need to clean it out on any error...
		return errors.New("You have a connection to this ip and port open already")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	} else {
		// Store the connection, it's valid
		bm.Lock()
		bm.dialedConnections[address] = conn
		bm.Unlock()
	}
	return nil
}

// Write a version of this that allows for automatic DialOuts, as well as one-shot connections that clean up afterward
func (bm *BuffManager) WriteTo(ip string, port string, data []byte, closeConnection bool) (int, error) {
	address := formatAddress(ip, port)
	// Get the connection if it's cached, or open a new one
	if _, ok := bm.dialedConnections[address]; ok != true {
		err := bm.dialOut(ip, port)
		if err != nil {
			// Error dialing out, cannot write
			// bail
			return 0, err
		}
	}
	// Calculate how big the message is, using a consistent header size. MAKE THIS CONFIGURABLE in some sane way
	toWriteLen := UInt16ToByteArray(uint16(len(data)), MessageSizeToBitLength(4096))
	// Append the size to the message, so now it has a header
	toWrite := append(toWriteLen, data...)
	bm.Lock()
	defer bm.Unlock()
	written, err := bm.dialedConnections[address].Write(toWrite)
	bm.dialedConnections[address].Close()
	delete(bm.dialedConnections, address)
	return written, err
}
