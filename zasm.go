package main

import (
	"fmt"
	"flag"
	"os"
)

func showHelp() {
	fmt.Fprintf(os.Stderr, "=========================================\n")
	fmt.Fprintf(os.Stderr, "                 Zasm-go                 \n")
	fmt.Fprintf(os.Stderr, "=========================================\n")
	fmt.Fprintf(os.Stderr, "Usage: go run main.go [options] srcfile outfile\n\n")
	fmt.Fprintf(os.Stderr, "Available Options:\n")
	flag.PrintDefaults() 
	fmt.Fprintf(os.Stderr, "\nExample:\n")
	fmt.Fprintf(os.Stderr, "  go run zasm-go.go test.asm output.86p\n")
	fmt.Fprintf(os.Stderr, "  zasm-go test.asm output.86p\n")
	fmt.Fprintf(os.Stderr, "=========================================\n")
}

func main() {
	inputfile := flag.String("inputfile", "", "z80 source code input file")
	outputfile := flag.String("outputfile", "z.out", "Name of output file to save assembled\nbinary filename.86p")
	outputstring := flag.Bool("outputstring", false, "Output as a string filename.86s")

	flag.Usage = showHelp

	flag.Parse()

	fmt.Println(*inputfile)
	fmt.Println(*outputfile)	
	fmt.Println(*outputstring)	
}
