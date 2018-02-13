package trie

import (
	"../../common"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// syncMemBatch is an in-memory buffer of successfully downloaded but not yet
// persisted data items.
type syncMemBatch struct {
	batch map[common.Hash][]byte // In-memory membatch of recently completed items
	order []common.Hash          // Order of completion to prevent out-of-order data loss
}

// request represents a scheduled or already in-flight state retrieval request.
type request struct {
	hash common.Hash // Hash of the node data content to retrieve
	data []byte      // Data content of the node, cached until all subtrees complete
	raw  bool        // Whether this is a raw entry (code) or a trie node

	parents []*request // Parent state nodes referencing this entry (notify all upon completion)
	depth   int        // Depth level within the trie the node is located to prioritise DFS
	deps    int        // Number of dependencies before allowed to commit this node

	callback TrieSyncLeafCallback // Callback to invoke if a leaf node it reached on this branch
}

// TrieSyncLeafCallback is a callback type invoked when a trie sync reaches a
// leaf node. It's used by state syncing to check if the leaf node requires some
// further data syncing.
type TrieSyncLeafCallback func(leaf []byte, parent common.Hash) error

// TrieSync is the main state trie synchronisation scheduler, which provides yet
// unknown trie hashes to retrieve, accepts node data associated with said hashes
// and reconstructs the trie step by step until all is done.
type TrieSync struct {
	database DatabaseReader           // Persistent database to check for existing entries
	membatch *syncMemBatch            // Memory buffer to avoid frequest database writes
	requests map[common.Hash]*request // Pending requests pertaining to a key hash
	queue    *prque.Prque             // Priority queue with the pending requests
}
