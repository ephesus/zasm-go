//Package passer parses the source and matches lines with
//generic lines in the .TAB file, after matches are found for all
//instructions, each instruction can be converted to the final binary form
//Using a "two-pass" strategy, the first pass finds .TAB matches and sizes
//and a second pass backfills addresses, finishing preparation for the binary generation
package passer

import (
	"fmt"
	"io"
)

func Pass() {
	fmt.Fprintf(io.Discard, "")
}
