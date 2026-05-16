package main

import (
	"flag"
	"fmt"
	"os"
	"zasm-go/passer"
)


const (
		colorReset  = "\033[0m"
		colorBlue   = "\033[38;5;75m"
		colorRed    = "\033[31m"
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

func errorExit(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s", colorRed, msg, colorReset)
	os.Exit(1)
}

func main() {
	var inputFile string
	var outputFile string
	var outputString bool

	inputfileFlag := flag.String("inputfile", "", "z80 source code input file")
	outputfileFlag := flag.String("outputfile", "", "Name of output file to save assembled\nbinary filename.86p")
	outputstringFlag := flag.Bool("outputstring", false, "Output as a string filename.86s")

	flag.Usage = showHelp

	flag.Parse()

	outputString = *outputstringFlag

	//if --innputfile is not set, use first positional param
	if *inputfileFlag != "" {
		inputFile = *inputfileFlag
	} else if len(flag.Args()) > 0 {
		inputFile = flag.Arg(0)
	} else {
		errorExit("Error: missing input file\n")
	}

	//if --outputfile is not set, use first or second positional param
	if *outputfileFlag != "" {
		outputFile = *outputfileFlag
	} else if len(flag.Args()) > 0 {
		outputFile = flag.Arg(len(flag.Args())-1)
	} 

	if (inputFile == outputFile) || (outputFile == "") {
		errorExit("Error: missing output filename\n")
	}

	passer.Pass()

	fmt.Println(outputString)
	fmt.Println(inputFile)
	fmt.Println(outputFile)
}




