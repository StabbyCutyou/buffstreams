// Package buffstreams provides a simple interface for creating and storing
// sockets that have a pre-defined set of behaviors, making them simple to use
// for consuming streaming protocol buffer messages over TCP
package buffstreams

// import (
// 	"errors"
// 	"net"
// 	"sync"
//
// 	"github.com/StabbyCutyou/buffstreams"
// )
//
// DefaultMaxMessageSize is the value that is used if a BuffManagerConfig indicates
// a MaxMessageSize of 0
const DefaultMaxMessageSize int = 4096

//
// // ErrAlreadyDialed represents the error where a caller has tried to dial the same
// // address more than once. Once a connection is opened, even if it's broken,
// // calling WriteTo will use the connection or create new ones using the existing
// // configuration
// const ErrAlreadyDialed = errors.New("This address is already dialed.")
//
// // ErrNotDialed represents the error where a caller has tried to write to an
// // address that they have not opened yet.
// const ErrNotDialed = errors.New("This address must be dialed before being written to.")
//
// // BuffManager represents the object used to govern interactions between tcp endpoints.
// // You can use it to read from and write to streaming or non-streaming TCP connections
// // and have it handle packaging data with a header describing the size of the data payload.
// // This is to make it easy to work with wire formats like ProtocolBuffers, which require
// // a custom-delimeter situation to be sent in a streaming fashion.
// type BuffManager struct {
// 	dialedConnections         map[string]*net.TCPConn
// 	listeningSockets          map[string]*buffstreams.BuffTCPListener
// 	listeningShutDownChannels map[string]chan (bool)
// 	dialerLock                *sync.RWMutex
// 	listenerLock              *sync.RWMutex
// }
//
// // New creates a new *BuffManager based on the provided BuffManagerConfig
// func New() *BuffManager {
// 	bm := &BuffManager{
// 		dialedConnections:         make(map[string]*net.TCPConn),
// 		listeningSockets:          make(map[string]*net.TCPListener),
// 		listeningShutDownChannels: make(map[string]chan (bool)),
// 		dialerLock:                &sync.RWMutex{},
// 		listenerLock:              &sync.RWMutex{},
// 	}
// 	return bm
// }
//

// FormatAddress is to cover the event that you want/need a programmtically correct way
// to format an address/port to use with StartListening or WriteTo
func FormatAddress(address string, port string) string {
	return address + ":" + port
}

//
// // StartListening is an asyncrhonous, non-blocking method. It begins listening on the given
// // port, and fire off a goroutine for every client connection it receives. That goroutine will
// // read the fixed header, then the message payload, and then invoke the povided ListenCallbacl.
// // In the event of an transport error, it will disconnect the client. It is the clients responsibility
// // to re-connect if needed.
// func (bm *BuffManager) StartListening(cfg BuffTCPListenerConfig) error {
// 	// Example BuffTCPListenerConfig
// 	// cfg := BuffTCPListenerConfig{
// 	// Address: FormatAddress("", port),
// 	// Callback: func([]bytes) error {return nil},
// 	// MaxMessageSize: 4096,
// 	// EnableLogging: False,
// 	// }
// 	socket := NewBuffTCPListener(cfg)
// 	bm.dialerLock.Lock()
// 	bm.listeningSockets[cfg.Address] = socket
// 	bm.dialerLock.Unlock()
//
// 	// By design, BuffTCPManager encourages laziness
// 	err := socket.StartListeningAsync()
// 	return err
// }
//
// // Dial must be called before attempting to write. This is because the BuffTCPWriter
// // need certain configuration information, which should be provided upfront. Once
// // the connection is open, there should be no need to check on it's status. WriteTo
// // will attempt to re-use or rebuild the connection using the existing connection if
// // any errors occur on a write. It will return the number of bytes written. While
// // the BuffTCPWriter makes every attempt to continue to send bytes until they are all
// // written, you should always check to make sure this number matches the bytes you
// // attempted to write, due to very exceptional cases.
// func (bm *BuffManager) Dial(cfg BuffTCPWriterConfig) error {
// 	// Grab the lock now, so nothing can sneak in
// 	// Release it before we exit early, or before the write
// 	bm.dialerLock.Lock()
// 	_, ok := bm.dialedConnections[cfg.Address]
//
// 	if ok {
// 		bm.dialerLock.Unlock()
// 		return ErrAlreadyDialed
// 	}
//
// 	buffConn := NewBuffTCPWriter(cfg)
//
// 	if err := buffConn.Open(); err != nil {
// 		bm.dialerLock.Unlock()
// 		return err
// 	}
// 	bm.dialedConnections[cfg.Address] = buffConn
// 	bm.dialerLock.Unlock()
// 	return nil
// }
//
// // WriteTo allows you to dial to a remote or local TCP endpoint, and send a series of
// // bytes as messages. Each array of bytes you pass in will be pre-pended with it's size
// // within the size of the pre-defined maximum message size. If the connection isn't open yet,
// // WriteTo will open it, and cache it. If for anyreason the connection breaks, it will be disposed
// // a. If not all bytes can be written,
// // WriteTo will keep trying until the full message is delivered, or the connection is broken.
// func (bm *BuffManager) WriteTo(address string, data []bytes) (int, err) {
// 	// Get the connection if it's cached, or open a new one
// 	bm.dialerLock.RLock()
// 	buffConn, ok := bm.dialedConnections[address]
// 	bm.dialerLock.RUnlock()
// 	if !ok {
// 		return 0, ErrNotDialed
// 	}
// 	bytesWritten, err := buffConn.Write(data)
// 	if err != nil {
// 		cfg := buffConn.CopyConfig()
//
// 	}
// 	return bytesWritten, err
// }
