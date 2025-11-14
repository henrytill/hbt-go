package types

import (
	"io"
)

type Parser interface {
	Parse(r io.Reader) (*Collection, error)
}

type Formatter interface {
	Format(w io.Writer, coll *Collection) error
}
