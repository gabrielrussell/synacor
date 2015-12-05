package main

import (
	"flag"
	"fmt"
	"os"
	"vm"
)

func main() {

	savedGame := flag.Bool("save", false, "load a saved vm instead of the program")
	saveOnEOF := flag.Bool("saveOnEOF", false, "save the game at input eof")
	debug := flag.Bool("debug", false, "run in debug mode")
	input := flag.String("in", "", "file to use as vm input")
	flag.Parse()
	var inFile io.Reader
	if input == "" {
		inFile = os.Stdin
	} else {
		inFile, err := os.Open(input)
		if err != nil {
			fmt.Printf("error opening input file %v %v\n", input, err)
			os.Exit(1)
		}
	}
	v := &vm.VM{
		Stdout:    os.Stdout,
		Stdin:     os.Stdin,
		SaveOnEOF: saveOnEOF,
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
	err = v.Run()
	if err != nil {
		fmt.Printf("program error %v\n", err)
		os.Exit(2)
	}
}
