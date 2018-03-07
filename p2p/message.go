package p2p

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/atmchain/atmapp/event"
	"github.com/atmchain/atmapp/p2p/discover"
	"github.com/atmchain/atmapp/rlp"
)

// Msg defines the structure of a p2p message.
//
// Note that a Msg can only be sent once since the Payload reader is
// consumed during sending. It is not possible to create a Msg and
// send it any number of times. If you want to reuse an encoded
// structure, encode the payload into a byte array and create a
// separate Msg with a bytes.Reader as Payload for each send.
type Msg struct {
	Code       uint64
	Size       uint32 // size of the paylod
	Payload    io.Reader
	ReceivedAt time.Time
}

// Decode parses the RLP content of a message into
// the given value, which must be a pointer.
//
// For the decoding rules, please see package rlp.
func (msg Msg) Decode(val interface{}) error {
	s := rlp.NewStream(msg.Payload, uint64(msg.Size))
	if err := s.Decode(val); err != nil {
		return newPeerError(errInvalidMsg, "(code %x) (size %d) %v", msg.Code, msg.Size, err)
	}
	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("msg #%v (%v bytes)", msg.Code, msg.Size)
}

// Discard reads any remaining payload data into a black hole.
func (msg Msg) Discard() error {
	_, err := io.Copy(ioutil.Discard, msg.Payload)
	return err
}

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	// WriteMsg sends a message. It will block until the message's
	// Payload has been consumed by the other end.
	//
	// Note that messages can be sent only once because their
	// payload reader is drained.
	WriteMsg(Msg) error
}

// MsgReadWriter provides reading and writing of encoded messages.
// Implementations should ensure that ReadMsg and WriteMsg can be
// called simultaneously from multiple goroutines.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

// Send writes an RLP-encoded message with the given code.
// data should encode as an RLP list.
func Send(w MsgWriter, msgcode uint64, data interface{}) error {
	size, r, err := rlp.EncodeToReader(data)
	if err != nil {
		return err
	}
	return w.WriteMsg(Msg{Code: msgcode, Size: uint32(size), Payload: r})
}

// SendItems writes an RLP with the given code and data elements.
// For a call such as:
//
//    SendItems(w, code, e1, e2, e3)
//
// the message payload will be an RLP list containing the items:
//
//    [e1, e2, e3]
//
func SendItems(w MsgWriter, msgcode uint64, elems ...interface{}) error {
	return Send(w, msgcode, elems)
}

// msgEventer wraps a MsgReadWriter and sends events whenever a message is sent
// or received
type msgEventer struct {
	MsgReadWriter

	feed     *event.Feed
	peerID   discover.NodeID
	Protocol string
}

// newMsgEventer returns a msgEventer which sends message events to the given
// feed
func newMsgEventer(rw MsgReadWriter, feed *event.Feed, peerID discover.NodeID, proto string) *msgEventer {
	return &msgEventer{
		MsgReadWriter: rw,
		feed:          feed,
		peerID:        peerID,
		Protocol:      proto,
	}
}

// ReadMsg reads a message from the underlying MsgReadWriter and emits a
// "message received" event
func (self *msgEventer) ReadMsg() (Msg, error) {
	msg, err := self.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	self.feed.Send(&PeerEvent{
		Type:     PeerEventTypeMsgRecv,
		Peer:     self.peerID,
		Protocol: self.Protocol,
		MsgCode:  &msg.Code,
		MsgSize:  &msg.Size,
	})
	return msg, nil
}

// WriteMsg writes a message to the underlying MsgReadWriter and emits a
// "message sent" event
func (self *msgEventer) WriteMsg(msg Msg) error {
	err := self.MsgReadWriter.WriteMsg(msg)
	if err != nil {
		return err
	}
	self.feed.Send(&PeerEvent{
		Type:     PeerEventTypeMsgSend,
		Peer:     self.peerID,
		Protocol: self.Protocol,
		MsgCode:  &msg.Code,
		MsgSize:  &msg.Size,
	})
	return nil
}

// Close closes the underlying MsgReadWriter if it implements the io.Closer
// interface
func (self *msgEventer) Close() error {
	if v, ok := self.MsgReadWriter.(io.Closer); ok {
		return v.Close()
	}
	return nil
}
