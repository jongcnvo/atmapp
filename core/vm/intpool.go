package vm

import (
	"fmt"
	"math/big"
)

const verifyPool = true

func verifyIntegerPool(ip *intPool) {
	for i, item := range ip.pool.data {
		if item.Cmp(checkVal) != 0 {
			panic(fmt.Sprintf("%d'th item failed aggressive pool check. Value was modified", i))
		}
	}
}

var checkVal = big.NewInt(-42)

const poolLimit = 256

// intPool is a pool of big integers that
// can be reused for all big.Int operations.
type intPool struct {
	pool *Stack
}

func newIntPool() *intPool {
	return &intPool{pool: newstack()}
}

func (p *intPool) get() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop()
	}
	return new(big.Int)
}
func (p *intPool) put(is ...*big.Int) {
	if len(p.pool.data) > poolLimit {
		return
	}

	for _, i := range is {
		// verifyPool is a build flag. Pool verification makes sure the integrity
		// of the integer pool by comparing values to a default value.
		if verifyPool {
			i.Set(checkVal)
		}

		p.pool.push(i)
	}
}
