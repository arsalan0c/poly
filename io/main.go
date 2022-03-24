package main

import (
	"fmt"
	"github.com/TimothyStiles/poly/io/gff"
)

// Started at 4:44
// 5:00 finished downloading all files
// 5:07 discovered big bug with gb file parsing (give this to poly folks)

func main() {
	// Parse Genbank file
	//ct := poly.GetCodonTable(11)
//	pass := "./test.gff"
	fail := "./ap.gff"
//       fail2 := "./spectest.gff" 
       sequence,_ := gff.Read(fail)
	for _, feature := range sequence.Features {
		if feature.Type == "CDS" {
			fmt.Println(feature)
		}
	}
	// Find associated uniprot numbers
}
