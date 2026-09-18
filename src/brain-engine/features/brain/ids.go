package brain

import (
	"crypto/rand"
	"encoding/hex"
)

// UUIDSource generates RFC 4122 version 4 UUID strings without adding an
// external dependency to the domain package.
type UUIDSource struct{}

func NewUUIDSource() IDSource {
	return UUIDSource{}
}

func (UUIDSource) NewBrainID() (BrainID, error) {
	id, err := newUUID()
	return BrainID(id), err
}

func (UUIDSource) NewRootID() (RootID, error) {
	id, err := newUUID()
	return RootID(id), err
}

func (UUIDSource) NewLayerID() (LayerID, error) {
	id, err := newUUID()
	return LayerID(id), err
}

func newUUID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
