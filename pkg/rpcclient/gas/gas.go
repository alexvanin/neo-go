/*
Package gas provides a convenience wrapper for GAS contract to use it via RPC.

GAS itself only has standard NEP-17 methods, so this package only contains its
hash and allows to create NEP-17 structures in an easier way.
*/
package gas

import (
	"github.com/nspcc-dev/neo-go/pkg/core/native/nativenames"
	"github.com/nspcc-dev/neo-go/pkg/core/state"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/nep17"
)

// Hash stores the hash of the native GAS contract.
var Hash = state.CreateNativeContractHash(nativenames.Gas)

// NewReader creates a NEP-17 reader for the GAS contract.
func NewReader(invoker nep17.Invoker) *nep17.TokenReader {
	return nep17.NewReader(invoker, Hash)
}

// New creates a NEP-17 contract instance for the native GAS contract.
func New(actor nep17.Actor) *nep17.Token {
	return nep17.New(actor, Hash)
}
