package accounts

// DerivationPath represents the computer friendly version of a hierarchical
// deterministic wallet account derivaion path.
//
// The BIP-32 spec https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
// defines derivation paths to be of the form:
//
//   m / purpose' / coin_type' / account' / change / address_index
//
// The BIP-44 spec https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
// defines that the `purpose` be 44' (or 0x8000002C) for crypto currencies, and
// SLIP-44 https://github.com/satoshilabs/slips/blob/master/slip-0044.md assigns
// the `coin_type` 60' (or 0x8000003C) to ATMChain.
//
// The root path for ATMChain is m/44'/60'/0'/0 according to the specification
// from https://github.com/atmchain/atmapp/issues, albeit it's not set in stone
// yet whether accounts should increment the last component or the children of
// that. We will go with the simpler approach of incrementing the last component.
type DerivationPath []uint32
