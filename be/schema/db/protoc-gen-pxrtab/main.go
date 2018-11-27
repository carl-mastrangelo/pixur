// protoc-grn-pxrtab is a protoc plugin for generating Pixur tables.
package main // import "pixur.org/pixur/be/schema/db/protoc-gen-pxrtab"

import (
	"log"
	"os"

	"pixur.org/pixur/be/schema/db/protoc-gen-pxrtab/generator"
)

func main() {
	if err := generator.New().Run(os.Stdout, os.Stdin); err != nil {
		log.Fatalln(err)
	}
}
