package nomaddemo

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// deterministicID derives a stable, UUID-shaped identifier from parts. Every
// simulated job/node/allocation ID in this package is computed this way
// (rather than generated once and stored) so that recomputing the demo's
// state from scratch on every request always yields the same IDs for the
// same logical entity — no persisted state or locking required.
func deterministicID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	sum := h.Sum(nil)
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

// deterministicPort derives a stable "dynamic" port number in Nomad's
// default dynamic port range (20000-32000) from parts.
func deterministicPort(parts ...string) int {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	sum := h.Sum(nil)
	const rangeSize = 32000 - 20000
	return 20000 + int(binary.BigEndian.Uint32(sum[:4])%rangeSize)
}
