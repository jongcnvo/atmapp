package state

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/atmchain/atmapp/common"
	"github.com/atmchain/atmapp/db"
)

func TestState(t *testing.T) {
	db, _ := db.NewMemDB()
	statedb, _ := New(common.Hash{}, NewDatabase(db))
	statedb.AddBalance(common.HexToAddress("7fb8994d4c97257efc888cb70eb2a0e93c9ad20e"), big.NewInt(100))
	b := statedb.GetBalance(common.HexToAddress("7fb8994d4c97257efc888cb70eb2a0e93c9ad20e"))
	fmt.Println(b.String())
	fmt.Println(statedb.IntermediateRoot(false).String())
}
