package main

import (
	"context"
	"flag"
	"log"
	"os"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/status"
)

var (
	infile  = flag.String("in", "", "The source file to make a thumbnail from")
	outfile = flag.String("out", "", "The destination file to write a thumbnail to")
)

func run(ctx context.Context, in, out string) status.S {
	fin, err := os.Open(in)
	if err != nil {
		return status.InvalidArgument(err, "can't open file")
	}
	defer fin.Close()

	im, sts := imaging.ReadImage(ctx, fin)
	if sts != nil {
		return sts
	}
	defer im.Close()

	thumb, sts := im.Thumbnail()
	if sts != nil {
		return sts
	}
	defer thumb.Close()

	fout, err := os.Create(out)
	if err != nil {
		return status.InvalidArgument(err, "can't create file")
	}
	defer fout.Close()

	if sts := thumb.Write(fout); sts != nil {
		return sts
	}
	return nil
}

func main() {
	flag.Parse()
	if sts := run(context.Background(), *infile, *outfile); sts != nil {
		log.Println(sts)
		os.Exit(1)
	}
}
