package internal

import (
	"io"

	"github.com/henrytill/hbt-go/internal/types"
)

type Formatter interface {
	Format(w io.Writer, collection *types.Collection) error
}
