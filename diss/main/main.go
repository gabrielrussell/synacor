package main

import (
	"flag"
	"fmt"
	"os"
	"vm"
)

func main() {

	savedGame := flag.Bool("save", false, "load a saved vm instead of the program")
	flag.Parse()
	v := &vm.VM{
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}
	fmt.Fprintf(os.Stderr, "flags %v\n", flag.Args())
	if len(flag.Args()) != 1 {
		fmt.Printf("usage vm <program.bin>\n")
		os.Exit(1)
	}
	var err error
	if *savedGame {
		err = v.LoadVM(flag.Arg(0))
	} else {
		err = v.Load(flag.Arg(0))
	}
	if err != nil {
		fmt.Printf("load failed %v\n", err)
	}
	diss := v.Disassemble(uint16(len(v.Mem)))
	fmt.Printf("%v", diss)
}
