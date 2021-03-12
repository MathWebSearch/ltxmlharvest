package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"

	"github.com/MathWebSearch/ltxmlharvest"
)

func main() {
	harvest := ltxmlharvest.Harvest(make([]ltxmlharvest.HarvestFragment, len(os.Args)-1))

	var err error
	for i, path := range os.Args[1:] {
		// open the file!
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		// do the harvest!
		harvest[i], err = ltxmlharvest.ReadXHTML(f)
		if err != nil {
			panic(err)
		}

		// set the id!
		harvest[i].ID = strconv.Itoa(i)
		harvest[i].URI = path

		f.Close()
	}

	// marshal it!
	node, err := xml.MarshalIndent(harvest, "", "   ")
	if err != nil {
		panic(err)
	}

	// and print!
	fmt.Println(string(node))
}
