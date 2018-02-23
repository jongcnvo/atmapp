package rpc

import (
	"gopkg.in/fatih/set.v0"
	"reflect"
	"sync"
)

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

// callback is a method callback which was registered in the server
type callback struct {
	rcvr        reflect.Value  // receiver of method
	method      reflect.Method // callback
	argTypes    []reflect.Type // input argument types
	hasCtx      bool           // method's first argument is a context (not included in argTypes)
	errPos      int            // err return idx, of -1 when method cannot return error
	isSubscribe bool           // indication if the callback is a subscription
}

// service represents a registered object
type service struct {
	name          string        // name for service
	typ           reflect.Type  // receiver type
	callbacks     callbacks     // registered handlers
	subscriptions subscriptions // available subscriptions/notifications
}

type serviceRegistry map[string]*service // collection of services
type callbacks map[string]*callback      // collection of RPC callbacks
type subscriptions map[string]*callback  // collection of subscription

// Server represents a RPC server
type Server struct {
	services serviceRegistry

	run      int32
	codecsMu sync.Mutex
	codecs   *set.Set
}

// NewServer will create a new server instance with no registered handlers.
func NewServer() *Server {
	server := &Server{
		services: make(serviceRegistry),
		codecs:   set.New(),
		run:      1,
	}

	// register a default service which will provide meta information about the RPC service such as the services and
	// methods it offers.
	//rpcService := &RPCService{server}
	//server.RegisterName(MetadataApi, rpcService)

	return server
}

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}
