// 29 december 2012
package main

import (
	"os"
	"io"
	"encoding/xml"
	"log"
)

// CHDs only have Name and SHA1
type ROM struct {
	Name	string		`xml:"name,attr"`
	Size		int64		`xml:"size,attr"`
	CRC32	string		`xml:"crc,attr"`
	SHA1	string		`xml:"sha1,attr"`
}

type Game struct {
	Name	string	`xml:"name,attr"`
	// TODO do I need romof, cloneof, etc.?
	ROMs	[]ROM	`xml:"rom"`
	CHDs	[]ROM	`xml:"disk"`
}

var mamexmlname string
var mamexml *xml.Decoder

func mamexmlreadfatal(err error) {
	log.Fatalf("could not read MAME XML file %s: %v\n", mamexmlname, err)
}

func openMAMEXML(filename string) {
	mamexmlname = filename
	f, err := os.Open(mamexmlname)
	if err != nil {
		log.Fatalf("could not open MAME XML file %s: %v\n", mamexmlname, err)
	}
	mamexml = xml.NewDecoder(f)
	for {		// read until the start of the first game
		t, err := mamexml.Token()
		if err != nil {
			mamexmlreadfatal(err)
		}
		if se, ok := t.(xml.StartElement); ok && se.Name.Local == "mame" {
			break
		}
	}
}

func getNextGame() (game Game, eof bool) {
	err := mamexml.Decode(&game)
	if err == io.EOF {
		eof = true
		return
	} else if err != nil {
		mamexmlreadfatal(err)
	}
	return
}

func main() {
	openMAMEXML(os.Args[1])
	for g, eof := getNextGame(); !eof; g, eof = getNextGame() {
		log.Printf("%+v\n", g)
	}
}
