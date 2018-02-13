package downloader

// dataPack is a data message returned by a peer for some query.
type dataPack interface {
	PeerId() string
	Items() int
	Stats() string
}

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)
