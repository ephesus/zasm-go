package passer

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type TabEntry struct {
	Operands string
	Opcode   string
	Size     int
	Encoding string
}

//each mnumonic like "ADD" will have a TabEntry for each pattern
//of operands, like B, C, or HL, BC, etc. (ADD will have 27 variations, from TASM80.TAB)
type EncodingTable map[string][]TabEntry

// DebugPrint prints every entry in the EncodingTable
func (table EncodingTable) DebugPrint() {
	fmt.Println("Table File hasmap:")

	mnemonics := make([]string, 0, len(table))
	for mnemonic := range table {
		mnemonics = append(mnemonics, mnemonic)
	}
	sort.Strings(mnemonics) //hashmaps aren't ordered, so sort first

	for _, mnemonic := range mnemonics {
		entries := table[mnemonic]
		fmt.Printf("%s (%d):\n", mnemonic, len(entries))
		for _, e := range entries {
			fmt.Printf("    operands=%-12s opcode=%-8s size=%d encoding=%s\n",
				e.Operands, e.Opcode, e.Size, e.Encoding)
		}
	}
}

//LoadTableFile loads up the TASM80.TAB file so the assembler knows which opcodes are
//available, but it's only been tested with the TASM80.TAB file so might need some
//more error checking to work with other .TAB files
func LoadTableFile(r io.Reader) (EncodingTable, error) {
	table := make(EncodingTable)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		mnemonic := fields[0]
		operands := strings.Trim(fields[1], `"`)
		hexcode := fields[2]
		size, _ := strconv.Atoi(fields[3])
		encoding := fields[4]

		entry := TabEntry{
			Operands: operands,
			Opcode:   hexcode,
			Size:     size,
			Encoding: encoding,
		}
		table[mnemonic] = append(table[mnemonic], entry)
	}
	return table, scanner.Err()
}
