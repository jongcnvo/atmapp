package bloombits

import (
	"context"
)

// Retrieval represents a request for retrieval task assignments for a given
// bit with the given number of fetch elements, or a response for such a request.
// It can also have the actual results set to be used as a delivery data struct.
//
// The contest and error fields are used by the light client to terminate matching
// early if an error is enountered on some path of the pipeline.
type Retrieval struct {
	Bit      uint
	Sections []uint64
	Bitsets  [][]byte

	Context context.Context
	Error   error
}
