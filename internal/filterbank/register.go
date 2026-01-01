// register.go - Filter bank factory registration.
package filterbank

import (
	aac "github.com/llehouerou/go-aac"
)

// init registers the filter bank factory with the aac package.
// This breaks the import cycle between aac and filterbank:
// - aac cannot import filterbank (due to: aac -> filterbank -> syntax -> aac)
// - filterbank can import aac (no reverse dependency creates a cycle)
//
// The factory pattern allows aac.Decoder to create filter banks without
// directly importing this package.
func init() {
	aac.RegisterFilterBankFactory(func(frameLength uint16) any {
		return NewFilterBank(frameLength)
	})
}
