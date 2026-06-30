package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"zasm-go/passer"
)

// DEBUG sections will be removed by compiler optimization when set to "false"
const DEBUG = true

var p *message.Printer = message.NewPrinter(language.English)

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
	fmt.Println(cfg.DoColor)
	if cfg.DoColor == true {
		formatStr = "❌ %s%s%s"
	} else {
		formatStr = "%s"
	}

	p.Fprintf(os.Stderr, formatStr, colorRed, msg, colorReset)
	os.Exit(1)
}

// Config is the global configuration settings
type Config struct {
	InputFile      string
	OutputFile     string
	TableFile      string
	OutputAsString bool
	DoColor        bool
}

//global instance of settings
var cfg Config

func parseFlags() Config {
	inputfileFlag := flag.String("inputfile", "", "z80 source code input file")
	outputfileFlag := flag.String("outputfile", "", "Name of output file to save assembled\nbinary filename.86p")
	tabfileFlag := flag.String("tabfile", "assets/TASM80.TAB", "Path to TASM-format encoding table")
	outputAsStringFlag := flag.Bool("outputstring", false, "Output as a string filename.86s")
	doColorFlag := flag.Bool("color", true, "Colorize output when available (1=on or 0=off)")

	flag.Usage = showHelp
	flag.Parse()

	cfg.TableFile = *tabfileFlag
	cfg.OutputAsString = *outputAsStringFlag
	cfg.DoColor = *doColorFlag

	// Get Input File
	// Flags are optional so figure it out by position if no flag is used
	if *inputfileFlag != "" {
		cfg.InputFile = *inputfileFlag
	} else if len(flag.Args()) > 0 {
		cfg.InputFile = flag.Arg(0)
	} else {
		errorExit("Error: missing input file\n")
	}

	// Get Output File
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
	// Not really needed since cfg is global
	return cfg
}

// debugPrint simply prints out log messages for debugging (sometimes with color)
func debugPrint(desc string, message any) {
	var formatStr string

	switch message.(type) {
	case bool:
		if cfg.DoColor == true {
			formatStr = " %s\t\t%t%s\n"
			p.Fprintf(os.Stdout, desc+formatStr, colorGreen, message, colorReset)
		} else {
			formatStr = " %t\n"
			p.Fprintf(os.Stdout, desc+formatStr, message)
		}
	case int:
		if cfg.DoColor == true {
			formatStr = " %s\t\t%d%s\n"
			p.Fprintf(os.Stdout, desc+formatStr, colorGreen, message, colorReset)
		} else {
			formatStr = " %d\n"
			p.Fprintf(os.Stdout, desc+formatStr, message)
		}
	default:
		//default to string
		if cfg.DoColor == true {
			formatStr = " %s\t\t%s%s\n"
			p.Fprintf(os.Stdout, desc+formatStr, colorGreen, message, colorReset)
		} else {
			formatStr = " %s\n"
			p.Fprintf(os.Stdout, desc+formatStr, message)
		}
	}
}

// main() is the entrypoint
func main() {
	//all global configuration is stored in the cfg
	var cfg = parseFlags()

	//open TAB file
	f, err := os.Open(cfg.TableFile)
	if err != nil {
		log.Fatalf("Error opening tab file %s: %v", cfg.TableFile, err)
	}
	defer f.Close()

	// Load the TAB file
	TableFile, err := passer.LoadTableFile(f)
	if err != nil {
		log.Fatalf("Error loading tab file %s: %v", cfg.TableFile, err)
	}

	// Read the source file
	src, err := os.ReadFile(cfg.InputFile)
	if err != nil {
		log.Fatalf("Error reading input file %s: %v", cfg.InputFile, err)
	}

	// Lex and parse the source into lines
	lexer := passer.NewLexer(string(src), cfg.InputFile, filepath.Dir(cfg.InputFile))
	parser := passer.NewParser(lexer, TableFile)
	lines := parser.Parse()

	// First pass: assign addresses and fill the symbol table
	if err := parser.Pass1(lines); err != nil {
		log.Fatalf("Pass 1 failed: %v", err)
	}

	//this will be optimized out by the compiler
	if DEBUG {
		fmt.Println("Debug INFO:")
		fmt.Println("----------------")
		debugPrint("outputfile:", cfg.OutputFile)
		debugPrint("inputfile:", cfg.InputFile)
		debugPrint("output as .86s:", cfg.OutputAsString)
		debugPrint("do color:", cfg.DoColor)

		// Print out the contents of the "lines" DS after Pass1
		passer.PrintLines(lines, parser.SymbolTable)

		// Print out the contents of the TableFile DS
		// TableFile.DebugPrint()
	}
}
