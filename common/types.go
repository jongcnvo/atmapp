package common

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"time"
)

// PrettyDuration is a pretty printed version of a time.Duration value that cuts
// the unnecessary precision off from the formatted textual representation.
type PrettyDuration time.Duration

const (
	//HashLength length of hash string
	HashLength = 32
	//AddressLength length of account address
	AddressLength = 20
)

// Common big integers often used
var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(0xff)
	Big257 = big.NewInt(257)
)

type StorageSize float64

func (self StorageSize) String() string {
	if self > 1000000 {
		return fmt.Sprintf("%.2f mB", self/1000000)
	} else if self > 1000 {
		return fmt.Sprintf("%.2f kB", self/1000)
	} else {
		return fmt.Sprintf("%.2f B", self)
	}
}

func (self StorageSize) Int64() int64 {
	return int64(self)
}

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// Address represents the 20 byte address
type Address [AddressLength]byte

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a sufficient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte

const (
	// BloomByteLength represents the number of bytes used in a header log bloom.
	BloomByteLength = 256

	// BloomBitLength represents the number of bits used in a header log bloom.
	BloomBitLength = 8 * BloomByteLength
)

// Bloom represents a 2048 bit bloom filter.
type Bloom [BloomByteLength]byte

var bytesT = reflect.TypeOf(Bytes(nil))

//SetBytes set the hash to the value of b. If b is larger than len(h), 'b' will be cropped (from the left).
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

//BytesToHash set the hash to the value of b
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

//HexToHash set the hash to the value of s
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

//Hex convert hash to hex string
func (h Hash) Hex() string { return Encode(h[:]) }

//Bytes convert hash to byte slice
func (h Hash) Bytes() []byte { return h[:] }

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

func EmptyHash(h Hash) bool {
	return h == Hash{}
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

// SetBytes sets the address to the value of b. If b is larger than len(a) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

//BytesToAddress set the address to the value to b
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

//HexToAddress set the address to the value of s
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

//Bytes convert address to byte slice
func (a Address) Bytes() []byte { return a[:] }

// MarshalText returns the hex representation of a.
func (a Address) MarshalText() ([]byte, error) {
	return Bytes(a[:]).MarshalText()
}

// UnmarshalText parses a hash in hex syntax.
func (a *Address) UnmarshalText(input []byte) error {
	return UnmarshalFixedText("Address", input, a[:])
}

// SetBytes sets the content of b to the given bytes.
// It panics if d is not of suitable size.
func (b *Bloom) SetBytes(d []byte) {
	if len(b) < len(d) {
		panic(fmt.Sprintf("bloom bytes too big %d %d", len(b), len(d)))
	}
	copy(b[BloomByteLength-len(d):], d)
}

// BytesToBloom converts a byte slice to a bloom filter.
// It panics if b is not of suitable size.
func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloom.SetBytes(b)
	return bloom
}

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Encode encodes b as a hex string with 0x prefix.
func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

// Bytes marshals/unmarshals as a JSON string with 0x prefix.
// The empty slice marshals as "0x".
type Bytes []byte

// MarshalText implements encoding.TextMarshaler
func (b Bytes) MarshalText() ([]byte, error) {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Bytes) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(bytesT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), bytesT)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *Bytes) UnmarshalText(input []byte) error {
	raw, err := checkText(input, true)
	if err != nil {
		return err
	}
	dec := make([]byte, len(raw)/2)
	if _, err = hex.Decode(dec, raw); err != nil {
		err = mapError(err)
	} else {
		*b = dec
	}
	return err
}

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

func errNonString(typ reflect.Type) error {
	return &json.UnmarshalTypeError{Value: "non-string", Type: typ}
}

func wrapTypeError(err error, typ reflect.Type) error {
	if _, ok := err.(*decError); ok {
		return &json.UnmarshalTypeError{Value: err.Error(), Type: typ}
	}
	return err
}

const badNibble = ^uint64(0)

func decodeNibble(in byte) uint64 {
	switch {
	case in >= '0' && in <= '9':
		return uint64(in - '0')
	case in >= 'A' && in <= 'F':
		return uint64(in - 'A' + 10)
	case in >= 'a' && in <= 'f':
		return uint64(in - 'a' + 10)
	default:
		return badNibble
	}
}

// UnmarshalFixedText decodes the input as a string with 0x prefix. The length of out
// determines the required input length. This function is commonly used to implement the
// UnmarshalText method for fixed-size types.
func UnmarshalFixedText(typname string, input, out []byte) error {
	raw, err := checkText(input, true)
	if err != nil {
		return err
	}
	if len(raw)/2 != len(out) {
		return fmt.Errorf("hex string has length %d, want %d for %s", len(raw), len(out)*2, typname)
	}
	// Pre-verify syntax before modifying out.
	for _, b := range raw {
		if decodeNibble(b) == badNibble {
			return ErrSyntax
		}
	}
	hex.Decode(out, raw)
	return nil
}

func bytesHave0xPrefix(input []byte) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func checkText(input []byte, wantPrefix bool) ([]byte, error) {
	if len(input) == 0 {
		return nil, nil // empty strings are allowed
	}
	if bytesHave0xPrefix(input) {
		input = input[2:]
	} else if wantPrefix {
		return nil, ErrMissingPrefix
	}
	if len(input)%2 != 0 {
		return nil, ErrOddLength
	}
	return input, nil
}

// LoadJSON reads the given file and unmarshals its content.
func LoadJSON(file string, val interface{}) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, val); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(content, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
		}
		return fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}
	return nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}

// Report gives off a warning requesting the user to submit an issue to the github tracker.
func Report(extra ...interface{}) {
	fmt.Fprintln(os.Stderr, "You've encountered a sought after, hard to reproduce bug. Please report this to the developers <3 https://github.com/atmchain/atmapp/issues")
	fmt.Fprintln(os.Stderr, extra...)

	_, file, line, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "%v:%v\n", file, line)

	debug.PrintStack()

	fmt.Fprintln(os.Stderr, "#### BUG! PLEASE REPORT ####")
}
