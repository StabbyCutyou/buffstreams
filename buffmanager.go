package buffstreams

import (
	"errors"
	"sync"
)

// ErrAlreadyOpened represents the error where a caller has tried to open the same
// ip / port address more than once.
var ErrAlreadyOpened = errors.New("A connection to this ip / port is already open.")

// ErrNotOpened represents the error where a caller has tried to use a socket to
// an address that they have not opened yet.
var ErrNotOpened = errors.New("A connection to this ip / port must be opened first.")

// BuffManager represents the object used to govern interactions between tcp endpoints.
// You can use it to read from and write to streaming or non-streaming TCP connections
// and have it handle packaging data with a header describing the size of the data payload.
// This is to make it easy to work with wire formats like ProtocolBuffers, which require
// a custom-delimeter situation to be sent in a streaming fashion.
type BuffManager struct {
	dialedConnections map[string]*BuffTCPWriter
	listeningSockets  map[string]*BuffTCPListener
	dialerLock        *sync.RWMutex
	listenerLock      *sync.Mutex
}

// NewBuffManager creates a new *BuffManager based on the provided BuffManagerConfig
func NewBuffManager() *BuffManager {
	bm := &BuffManager{
		dialedConnections: make(map[string]*BuffTCPWriter),
		listeningSockets:  make(map[string]*BuffTCPListener),
		dialerLock:        &sync.RWMutex{},
		listenerLock:      &sync.Mutex{},
	}
	return bm
}

// StartListening is an asyncrhonous, non-blocking method. It begins listening on the given
// port, and fire off a goroutine for every client connection it receives. That goroutine will
// read the fixed header, then the message payload, and then invoke the povided ListenCallbacl.
// In the event of an transport error, it will disconnect the client. It is the clients responsibility
// to re-connect if needed.
func (bm *BuffManager) StartListening(cfg BuffTCPListenerConfig) error {
	// Example BuffTCPListenerConfig
	// cfg := BuffTCPListenerConfig{
	//   Address: FormatAddress("", port),
	//   Callback: func([]bytes) error {return nil},
	//   MaxMessageSize: 4096,
	//   EnableLogging: False,
	// }

	bm.listenerLock.Lock()
	defer bm.listenerLock.Unlock()
	if _, ok := bm.listeningSockets[cfg.Address]; ok == true {
		return ErrAlreadyOpened
	}

	btl, err := ListenBuffTCP(cfg)
	if err != nil {
		return err
	}
	bm.listeningSockets[cfg.Address] = btl
	// By design, BuffTCPManager encourages laziness
	return btl.StartListeningAsync()
}

// CloseListener lets you send a signal to a BuffTCPListener that tells it to
// stop accepting new requests. It will finish any requests in flight.
func (bm *BuffManager) CloseListener(address string) error {
	bm.listenerLock.Lock()
	defer bm.listenerLock.Unlock()
	if btl, ok := bm.listeningSockets[address]; ok == true {
		btl.Close()
		return nil
	}
	// If it wasn't opened, we hit this condition - return error
	return ErrNotOpened
}

// Dial must be called before attempting to write. This is because the BuffTCPWriter
// need certain configuration information, which should be provided upfront. Once
// the connection is open, there should be no need to check on it's status. WriteTo
// will attempt to re-use or rebuild the connection using the existing connection if
// any errors occur on a write. It will return the number of bytes written. While
// the BuffTCPWriter makes every attempt to continue to send bytes until they are all
// written, you should always check to make sure this number matches the bytes you
// attempted to write, due to very exceptional cases.
func (bm *BuffManager) Dial(cfg BuffTCPWriterConfig) error {
	bm.dialerLock.Lock()
	defer bm.dialerLock.Unlock()
	if _, ok := bm.dialedConnections[cfg.Address]; ok {
		return ErrAlreadyOpened
	}

	btw, err := DialBuffTCP(cfg)
	if err != nil {
		return err
	}
	bm.dialedConnections[cfg.Address] = btw
	return nil
}

// CloseWriter lets you send a signal to a BuffTCPWriter that tells it to
// stop accepting new requests. It will finish any requests in flight.
func (bm *BuffManager) CloseWriter(address string) error {
	bm.dialerLock.Lock()
	defer bm.dialerLock.Unlock()
	if btw, ok := bm.dialedConnections[address]; ok == true {
		return btw.Close()
	}
	// If it wasn't opened, we hit this condition - return error
	return ErrNotOpened
}

// WriteTo allows you to dial to a remote or local TCP endpoint, and send a series of
// bytes as messages. Each array of bytes you pass in will be pre-pended with it's size
// within the size of the pre-defined maximum message size. If the connection isn't open yet,
// WriteTo will open it, and cache it. If for anyreason the connection breaks, it will be disposed
// a. If not all bytes can be written,
// WriteTo will keep trying until the full message is delivered, or the connection is broken.
func (bm *BuffManager) WriteTo(address string, data []byte) (int, error) {
	// Get the connection if it's cached, or open a new one
	bm.dialerLock.RLock()
	btw, ok := bm.dialedConnections[address]
	bm.dialerLock.RUnlock()
	if !ok {
		return 0, ErrNotOpened
	}
	bytesWritten, err := btw.Write(data)
	if err != nil {
		btw.Reopen()

	}
	return bytesWritten, err
}
