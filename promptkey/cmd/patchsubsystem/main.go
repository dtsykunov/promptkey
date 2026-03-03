// patchsubsystem patches a Windows PE binary's subsystem field to CONSOLE (3).
// Wails forces -H windowsgui in its linker flags, which always wins over any
// user-supplied -H console. This tool edits the built .exe directly.
//
// Usage: go run ./cmd/patchsubsystem <path-to.exe>
package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: patchsubsystem <exe>")
		os.Exit(1)
	}
	path := os.Args[1]
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// PE header offset is stored as uint32 at DOS offset 0x3C.
	var peOff uint32
	if _, err := f.Seek(0x3c, 0); err != nil {
		fmt.Fprintf(os.Stderr, "seek 0x3c: %v\n", err)
		os.Exit(1)
	}
	if err := binary.Read(f, binary.LittleEndian, &peOff); err != nil {
		fmt.Fprintf(os.Stderr, "read pe offset: %v\n", err)
		os.Exit(1)
	}

	// Subsystem is at: PE sig (4) + COFF header (20) + 68 bytes into Optional Header.
	subsysOff := int64(peOff) + 92
	if _, err := f.Seek(subsysOff, 0); err != nil {
		fmt.Fprintf(os.Stderr, "seek subsystem: %v\n", err)
		os.Exit(1)
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(3)); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("patched %s → SUBSYSTEM_CONSOLE\n", path)
}
