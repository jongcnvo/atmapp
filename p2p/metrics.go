package p2p

import (
	"net"
)

var (
//ingressConnectMeter = metrics.NewMeter("p2p/InboundConnects")
//ingressTrafficMeter = metrics.NewMeter("p2p/InboundTraffic")
//egressConnectMeter  = metrics.NewMeter("p2p/OutboundConnects")
//egressTrafficMeter  = metrics.NewMeter("p2p/OutboundTraffic")
)

// meteredConn is a wrapper around a network TCP connection that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	*net.TCPConn // Network connection to wrap with metering
}

// newMeteredConn creates a new metered connection, also bumping the ingress or
// egress connection meter. If the metrics system is disabled, this function
// returns the original object.
func newMeteredConn(conn net.Conn, ingress bool) net.Conn {
	// Short circuit if metrics are disabled
	//if !metrics.Enabled {
	//	return conn
	//}
	// Otherwise bump the connection counters and wrap the connection
	//if ingress {
	//	ingressConnectMeter.Mark(1)
	//} else {
	//	egressConnectMeter.Mark(1)
	//}
	return &meteredConn{conn.(*net.TCPConn)}
}
