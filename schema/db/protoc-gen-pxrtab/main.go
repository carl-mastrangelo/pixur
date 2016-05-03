package main

import (
	"log"
	"os"

	"pixur.org/pixur/schema/db/protoc-gen-pxrtab/generator"
)

func main() {
	gen := new(generator.Generator)
	if err := gen.Run(os.Stdout, os.Stdin); err != nil {
		log.Fatalln(err)
	}
}
