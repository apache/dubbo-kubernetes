package hash

import (
	"encoding/hex"

	"github.com/cespare/xxhash/v2"
)

type Hash interface {
	Write(p []byte) (n int)
	WriteString(s string) (n int)
	Sum() string
	Sum64() uint64
}

type instance struct {
	hash *xxhash.Digest
}

var _ Hash = &instance{}

func New() Hash {
	return &instance{
		hash: xxhash.New(),
	}
}

func (i *instance) Write(p []byte) (n int) {
	n, _ = i.hash.Write(p)
	return
}

func (i *instance) WriteString(s string) (n int) {
	n, _ = i.hash.WriteString(s)
	return
}

func (i *instance) Sum64() uint64 {
	return i.hash.Sum64()
}

func (i *instance) Sum() string {
	sum := i.hash.Sum(nil)
	return hex.EncodeToString(sum)
}
