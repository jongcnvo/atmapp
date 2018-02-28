package atm

import (
	"compress/gzip"
	"io"
	"os"
	"strings"

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

// ExportChain exports the current blockchain into a local file.
func (api *PrivateAdminAPI) ExportChain(file string) (bool, error) {
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer out.Close()

	var writer io.Writer = out
	if strings.HasSuffix(file, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	// Export the blockchain
	if err := api.atm.BlockChain().Export(writer); err != nil {
		return false, err
	}
	return true, nil
}
