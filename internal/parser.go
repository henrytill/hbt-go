package internal

import (
	"io"

	"github.com/henrytill/hbt-go/internal/types"
)

type Parser interface {
	Parse(r io.Reader) (*types.Collection, error)
}
