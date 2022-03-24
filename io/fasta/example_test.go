package fasta_test

import (
	"fmt"

	"testing"

	"github.com/TimothyStiles/poly/io/fasta"
)

// This example shows how to open a file with the fasta parser. The sequences
// within that file can then be analyzed further with different software.
func TestExample_basic(t *testing.T) {
	fastas, _ := fasta.Read("data/base.fasta")
	fmt.Println(fastas[1].Sequence)
	// Output: ADQLTEEQIAEFKEAFSLFDKDGDGTITTKELGTVMRSLGQNPTEAELQDMINEVDADGNGTIDFPEFLTMMARKMKDTDSEEEIREAFRVFDKDGNGYISAAELRHVMTNLGEKLTDEEVDEMIREADIDGDGQVNYEEFVQMMTAK*
}
