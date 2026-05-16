package main

import (
	"fmt"
	"flag"
	"os"
	"zasm-go/passer"
)


const (
		colorReset  = "\033[0m"
		colorBlue   = "\033[38;5;75m"
		colorGray   = "\033[38;5;243m"
		colorOrange = "\033[38;5;215m"
		colorPurple = "\033[38;5;141m"
		colorWhite  = "\033[38;5;255m"
		colorGreen  = "\033[38;5;150m"
		bold        = "\033[1m"
		dim         = "\033[2m"
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

	passer.Pass()
}
