package atm

import (
	"github.com/atmchain/atmapp/common"
)

// PublicATMPI provides an API to access ATMChain full node-related
// information.
type PublicATMAPI struct {
	e *ATM
}

// NewPublicATMAPI creates a new ATMChain protocol API for full nodes.
func NewPublicATMAPI(e *ATM) *PublicATMAPI {
	return &PublicATMAPI{e}
}

// ATMrbase is the address that mining rewards will be send to
func (api *PublicATMAPI) ATMbase() (common.Address, error) {
	return api.e.ATMbase()
}

// PrivateAdminAPI is the collection of ATMChain full node-related APIs
// exposed over the private admin endpoint.
type PrivateAdminAPI struct {
	atm *ATM
}

// NewPrivateAdminAPI creates a new API definition for the full node private
// admin methods of the ATMChain service.
func NewPrivateAdminAPI(atm *ATM) *PrivateAdminAPI {
	return &PrivateAdminAPI{atm: atm}
}
