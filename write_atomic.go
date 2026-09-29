package insyra

import (
	"io"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// writeFileAtomically writes path through a temporary file renamed into place;
// see utils.WriteFileAtomically.
func writeFileAtomically(path string, write func(w io.Writer) error) error {
	return utils.WriteFileAtomically(path, write)
}
