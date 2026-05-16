package main

import (
	"flag"
	"os"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"zasm-go/passer"
)

const DEBUG = true

var p *message.Printer = message.NewPrinter(language.English)

//Config is the global configuration settings
type Config struct {
	InputFile    string
	OutputFile   string
	OutputString bool
	DoColor      bool
}
var cfg Config

func parseFlags() Config {
	inputfileFlag := flag.String("inputfile", "", "z80 source code input file")
	outputfileFlag := flag.String("outputfile", "", "Name of output file to save assembled\nbinary filename.86p")
	outputstringFlag := flag.Bool("outputstring", false, "Output as a string filename.86s")
	doColorFlag := flag.Bool("color", true, "Colorize output when available")

	flag.Usage = showHelp
	flag.Parse()

	cfg.OutputString = *outputstringFlag
	cfg.DoColor = *doColorFlag

	// Resolve Input File
	if *inputfileFlag != "" {
		cfg.InputFile = *inputfileFlag
	} else if len(flag.Args()) > 0 {
		cfg.InputFile = flag.Arg(0)
	} else {
		errorExit("Error: missing input file\n")
	}

	// Resolve Output File
	if *outputfileFlag != "" {
		cfg.OutputFile = *outputfileFlag
	} else if len(flag.Args()) > 0 {
		cfg.OutputFile = flag.Arg(len(flag.Args()) - 1)
	}

	// Post-parsing relationship validations
	if (cfg.InputFile == cfg.OutputFile) || (cfg.OutputFile == "") {
		errorExit("Error: missing or invalid output filename\n")
	}

	// Return the populated config object back to main
	return cfg
}

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
	p.Fprintf(os.Stderr, "=========================================\n")
	p.Fprintf(os.Stderr, "                 Zasm-go                 \n")
	p.Fprintf(os.Stderr, "=========================================\n")
	p.Fprintf(os.Stderr, "Usage: go run main.go [options] srcfile outfile\n\n")
	p.Fprintf(os.Stderr, "Available Options:\n")
	flag.PrintDefaults() 
	p.Fprintf(os.Stderr, "\nExample:\n")
	p.Fprintf(os.Stderr, "  go run zasm-go.go test.asm output.86p\n")
	p.Fprintf(os.Stderr, "  zasm-go test.asm output.86p\n")
	p.Fprintf(os.Stderr, "=========================================\n")
}

func errorExit(msg string) {
	var formatStr string
	if cfg.DoColor {
		formatStr = "❌ %s%s%s"	
	} else {
		formatStr = "%s"
	}

	p.Fprintf(os.Stderr, formatStr, colorRed, msg, colorReset)
	os.Exit(1)
}

func main() {
	var cfg = parseFlags()

	passer.Pass()

	if DEBUG {
		p.Fprintf(os.Stdout, "%s%s%s\n", colorGreen, cfg.OutputFile, colorReset)
		p.Fprintf(os.Stdout, "%s%s%s\n", colorGreen, cfg.InputFile, colorReset)
		p.Fprintf(os.Stdout, "%s%t%s\n", colorGreen, cfg.OutputString, colorReset)
	}
}




