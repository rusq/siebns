package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rusq/siebns"
)

const version = "2.0.0"

func main() {
	fmt.Printf("Siebnsfix %s - fix checksum in Siebel Gateway file\n", version)
	if len(os.Args) < 2 {
		fmt.Printf("\nUsage: %s <siebns.dat>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	ns, err := siebns.Open(os.Args[1])
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer ns.Close()

	if !ns.IsHeaderCorrect() {
		log.Printf("file %s:  OK:  no correction needed.\n", ns.Name())
		return
	}

	log.Printf("file %s:  correction needed.\n", ns.Name())
	wrote, err := ns.FixSize()
	if err != nil {
		log.Fatalf("error writing to file:  %s\n", err)
	}
	log.Printf("file %s:  OK: updated %d bytes.\n", ns.Name(), wrote)

}
