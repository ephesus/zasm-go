package passer

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type TabEntry struct {
	Operands string
	Opcode   string
	Size     int
	Encoding string
}

type EncodingTable map[string][]TabEntry

func LoadTabFile(r io.Reader) (EncodingTable, error) {
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
