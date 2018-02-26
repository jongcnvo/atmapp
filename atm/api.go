package atm

// PrivateAdminAPI is the collection of Ethereum full node-related APIs
// exposed over the private admin endpoint.
type PrivateAdminAPI struct {
	atm *ATM
}

// NewPrivateAdminAPI creates a new API definition for the full node private
// admin methods of the ATMChain service.
func NewPrivateAdminAPI(atm *ATM) *PrivateAdminAPI {
	return &PrivateAdminAPI{atm: atm}
}
