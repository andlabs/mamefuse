// 29 december 2012
package main

import (
	"os"
	"encoding/xml"
	"strings"
	"io"
	"log"
)

const nodump = "nodump"	// for ROM.Status

// CHDs only have Name, SHA1, and Status
type ROM struct {
	Name	string		`xml:"name,attr"`
	Size		uint32		`xml:"size,attr"`		// uint32 because that's what archive/zip.FIleHeader.UncompressedSize is
	CRC32	string		`xml:"crc,attr"`
	SHA1	string		`xml:"sha1,attr"`
	Status	string		`xml:"status,attr"`
}

type Game struct {
	Name	string	`xml:"name,attr"`
	CloneOf	string	`xml:"cloneof,attr"`
	ROMOf	string	`xml:"romof,attr"`
	// TODO do I need sampleof?
	ROMs	[]ROM	`xml:"rom"`
	CHDs	[]ROM	`xml:"disk"`

	// prepared by getGames()
	Parents	[]string	// [CloneOf, ROMOf] but only if either is specified and no repeats; avoids code duplication in check.go

	// prepared by Game.Find()
	Found	bool
	ROMLoc	string
	CHDLoc	map[string]string
}

var games = map[string]*Game{}

func getGames(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("could not open MAME XML file %s: %v", filename, err)
	}
	defer f.Close()

	mamexml := xml.NewDecoder(f)

	// skip to the first game
findFirst:
	for {
		t, err := mamexml.Token()
		if err != nil {
			log.Fatalf("error finding first game in MAME XML file %s: %v", filename, err)
		}
		switch e := t.(type) {
		case xml.StartElement:
			if strings.ToLower(e.Name.Local) == "mame" {
				break findFirst
			}
		}
	}

	// now read everything
	for {
		this := new(Game)
		err = mamexml.Decode(this)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("error reading game from MAME XML file %s: %v", filename, err)
		}
		games[this.Name] = this
		if this.CloneOf != "" {
			this.Parents = append(this.Parents, this.CloneOf)
		}
		if this.ROMOf != "" && this.ROMOf != this.CloneOf {
			this.Parents = append(this.Parents, this.ROMOf)
		}
	}
}
