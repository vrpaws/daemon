package main

import (
	"log"

	lib "vrc-moments/pkg"
	"vrc-moments/pkg/vrc"
)

func main() {
	files, err := lib.ExpandPatterns(false, true, "~/Pictures/VRChat/***.png")
	if err != nil {
		log.Fatal(err)
	}

	if len(files) == 0 {
		log.Fatal("No files found.")
	}

	for _, file := range files {
		log.Println("Reading from file:", file)
		data, err := vrc.GetVRCXDataFromFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}
		log.Printf("%+v\n", data)
		break
	}
}
