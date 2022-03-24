/*
Package gff provides gff parsers and writers.

GFF stands for "general feature format". It is an alternative to GenBank for
storing data about genomic sequences. While not often used in synthetic biology
research, it is more commonly used in bioinformatics for digesting features of
genomic sequences.

This package provides a parser and writer to convert between the gff file
format and the more general poly.Sequence struct.
*/
package gff

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"

	"lukechampine.com/blake3"

	"github.com/TimothyStiles/poly/transform"
)

// Gff is a struct that represents a gff file.
type Gff struct {
	Meta     Meta
	Features []Feature // will need a GetFeatures interface to standardize
	Sequence string
}

// Meta holds meta information about a gff file.
type Meta struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Version              string   `json:"gff_version"`
	RegionStart          int      `json:"region_start"`
	RegionEnd            int      `json:"region_end"`
	Size                 int      `json:"size"`
	SequenceHash         string   `json:"sequence_hash"`
	SequenceHashFunction string   `json:"hash_function"`
	CheckSum             [32]byte `json:"checkSum"` // blake3 checksum of the parsed file itself. Useful for if you want to check if incoming genbank/gff files are different.
}

// Feature is a struct that represents a feature in a gff file.
type Feature struct {
	Name           string            `json:"name"`
	Source         string            `json:"source"`
	Type           string            `json:"type"`
	Score          string            `json:"score"`
	Strand         string            `json:"strand"`
	Phase          string            `json:"phase"`
	Attributes     map[string]string `json:"attributes"`
	Location       Location          `json:"location"`
	ParentSequence *Gff              `json:"-"`
}

// Location is a struct that represents a location in a gff file.
type Location struct {
	Start             int        `json:"start"`
	End               int        `json:"end"`
	Complement        bool       `json:"complement"`
	Join              bool       `json:"join"`
	FivePrimePartial  bool       `json:"five_prime_partial"`
	ThreePrimePartial bool       `json:"three_prime_partial"`
	SubLocations      []Location `json:"sub_locations"`
}

//AddFeature takes a feature and adds it to the Gff struct.
func (sequence *Gff) AddFeature(feature *Feature) error {
	feature.ParentSequence = sequence
	var featureCopy Feature = *feature
	sequence.Features = append(sequence.Features, featureCopy)
	return nil
}

// GetSequence takes a feature and returns a sequence string for that feature.
func (feature Feature) GetSequence() (string, error) {
	return getFeatureSequence(feature, feature.Location)
}

// getFeatureSequence takes a feature and location object and returns a sequence string.
func getFeatureSequence(feature Feature, location Location) (string, error) {
	var sequenceBuffer bytes.Buffer
	var sequenceString string
	parentSequence := feature.ParentSequence.Sequence

	if len(location.SubLocations) == 0 {
		sequenceBuffer.WriteString(parentSequence[location.Start:location.End])
	} else {

		for _, subLocation := range location.SubLocations {
			sequence, err := getFeatureSequence(feature, subLocation)
			if err != nil {
				return sequenceBuffer.String(), err
			}
			sequenceBuffer.WriteString(sequence)
		}
	}

	// reverse complements resulting string if needed.
	if location.Complement {
		sequenceString = transform.ReverseComplement(sequenceBuffer.String())
	} else {
		sequenceString = sequenceBuffer.String()
	}

	return sequenceString, nil
}

// Parse Takes in a string representing a gffv3 file and parses it into an Sequence object.
func Parse(file []byte) (Gff, error) {
	gffString := string(file)
	gff := Gff{}
	// Add the CheckSum to sequence (blake3)
	gff.Meta.CheckSum = blake3.Sum256(file)

	meta := Meta{}

	lines := strings.Split(gffString, "\n")
	versionString := lines[0]

	meta.Version = strings.Split(versionString, " ")[1]

	var sequenceBuffer bytes.Buffer
	fastaFlag := false
	for _, line := range lines {
		if line == "##FASTA" {
			fastaFlag = true
		} else if strings.HasPrefix(line, "##sequence-region") {
			regionStringArray := strings.Split(line, " ")
			meta.Name = regionStringArray[1] // Formally region name, but changed to name here for generality/interoperability.
			meta.RegionStart, _ = strconv.Atoi(regionStringArray[2])
			meta.RegionEnd, _ = strconv.Atoi(regionStringArray[3])
			meta.Size = meta.RegionEnd - meta.RegionStart
		} else if len(line) == 0 {
			continue
		} else if line[0:2] == "##" {
			continue
		} else if line[0:1] == "#" { // single hash sign signifies a human readable comment
			continue
		} else if fastaFlag && line[0:1] != ">" {
			// sequence.Sequence = sequence.Sequence + line
			sequenceBuffer.WriteString(line)
		} else if fastaFlag && line[0:1] == ">" {
			gff.Meta.Description = line
		} else {
			record := Feature{}
			fmt.Println(line)
			fields := strings.Split(line, "\t")
			record.Name = fields[0]
			record.Source = fields[1]
			record.Type = fields[2]

			// Indexing starts at 1 for gff so we need to shift down for Sequence 0 index.
			record.Location.Start, _ = strconv.Atoi(fields[3])
			record.Location.Start--
			record.Location.End, _ = strconv.Atoi(fields[4])

			record.Score = fields[5]
			record.Strand = fields[6]
			record.Phase = fields[7]
			record.Attributes = make(map[string]string)
			attributes := fields[8]
			// var eqIndex int
			attributeSlice := strings.Split(attributes, ";")

			for _, attribute := range attributeSlice {
				attributeSplit := strings.Split(attribute, "=")
				key := attributeSplit[0]
				value := attributeSplit[1]
				record.Attributes[key] = value
			}
			_ = gff.AddFeature(&record)
		}
	}
	gff.Sequence = sequenceBuffer.String()
	gff.Meta = meta

	return gff, nil
}

// Build takes an Annotated sequence and returns a byte array representing a gff to be written out.
func Build(sequence Gff) ([]byte, error) {
	var gffBuffer bytes.Buffer

	var versionString string
	if sequence.Meta.Version != "" {
		versionString = "##gff-version " + sequence.Meta.Version + "\n"
	} else {
		versionString = "##gff-version 3 \n"
	}
	gffBuffer.WriteString(versionString)

	var regionString string
	var name string
	var start string
	var end string

	if sequence.Meta.Name != "" {
		name = sequence.Meta.Name
	} else {
		name = "Sequence"
	}

	if sequence.Meta.RegionStart != 0 {
		start = strconv.Itoa(sequence.Meta.RegionStart)
	} else {
		start = "1"
	}

	end = strconv.Itoa(sequence.Meta.RegionEnd)

	regionString = "##sequence-region " + name + " " + start + " " + end + "\n"
	gffBuffer.WriteString(regionString)

	for _, feature := range sequence.Features {
		var featureString string
		var featureSource string
		if feature.Source != "" {
			featureSource = feature.Source
		} else {
			featureSource = "feature"
		}

		var featureType string
		if feature.Type != "" {
			featureType = feature.Type
		} else {
			featureType = "unknown"
		}

		// Indexing starts at 1 for gff so we need to shift up from Sequence 0 index.
		featureStart := strconv.Itoa(feature.Location.Start + 1)
		featureEnd := strconv.Itoa(feature.Location.End)

		featureScore := feature.Score
		featureStrand := string(feature.Strand)
		featurePhase := feature.Phase
		var featureAttributes string

		keys := make([]string, 0, len(feature.Attributes))
		for key := range feature.Attributes {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			attributeString := key + "=" + feature.Attributes[key] + ";"
			featureAttributes += attributeString
		}

		if len(featureAttributes) > 0 {
			featureAttributes = featureAttributes[0 : len(featureAttributes)-1]
		}
		TAB := "\t"
		featureString = feature.Name + TAB + featureSource + TAB + featureType + TAB + featureStart + TAB + featureEnd + TAB + featureScore + TAB + featureStrand + TAB + featurePhase + TAB + featureAttributes + "\n"
		gffBuffer.WriteString(featureString)
	}

	gffBuffer.WriteString("###\n")
	gffBuffer.WriteString("##FASTA\n")
	gffBuffer.WriteString(">" + sequence.Meta.Name + "\n")

	for letterIndex, letter := range sequence.Sequence {
		letterIndex++
		if letterIndex%70 == 0 && letterIndex != 0 && letterIndex != sequence.Meta.RegionEnd {
			gffBuffer.WriteRune(letter)
			gffBuffer.WriteString("\n")
		} else {
			gffBuffer.WriteRune(letter)
		}
	}
	gffBuffer.WriteString("\n")
	return gffBuffer.Bytes(), nil
}

// Read takes in a filepath for a .gffv3 file and parses it into an Annotated poly.Sequence struct.
func Read(path string) (Gff, error) {
	fmt.Println("Going to print!")
	file, _ := ioutil.ReadFile(path)
	sequence, err := Parse(file)
	if err != nil {
		return Gff{}, err
	}
	return sequence, nil
}

// Write takes an poly.Sequence struct and a path string and writes out a gff to that path.
func Write(sequence Gff, path string) error {
	gff, err := Build(sequence)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, gff, 0644)
	return err
}
