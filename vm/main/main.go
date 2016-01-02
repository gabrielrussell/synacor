package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"vm"
)

func main() {

	savedGame := flag.Bool("save", false, "load a saved vm instead of the program")
	saveOnEOF := flag.Bool("saveOnEOF", false, "save the game at input eof")
	metadataFile := flag.String("metadata", ".metadata", "file of general metadata to update")
	debug := flag.Bool("debug", false, "run in debug mode")
	input := flag.String("in", "", "file to use as vm input")
	flag.Parse()
	var err error
	var inFile io.Reader
	if input != nil {
		inFile = os.Stdin
	} else {
		inFile, err = os.Open(*input)
		if err != nil {
			fmt.Printf("error opening input file %v %v\n", input, err)
			os.Exit(1)
		}
	}
	v := &vm.VM{
		Stdout:       os.Stdout,
		Stdin:        bufio.NewReader(inFile),
		SaveOnEOF:    *saveOnEOF,
		Debugging:    *debug,
		ControlChan:  make(chan string),
		BreakOps:     make(map[uint16]bool),
		Break:        make(map[uint16]bool),
		MetadataFile: *metadataFile,
	}
	fmt.Fprintf(os.Stderr, "flags %v\n", flag.Args())
	if len(flag.Args()) != 1 {
		fmt.Printf("usage vm <program.bin>\n")
		os.Exit(1)
	}
	if *savedGame {
		err = v.LoadVM(flag.Arg(0))
	} else {
		err = v.Load(flag.Arg(0))
	}
	v.LoadMetadata()
	if err != nil {
		fmt.Printf("load failed %v\n", err)
	}
	go v.Run()
	if v.Debugging {
		fmt.Printf("starting debugger\n")
		err = v.Debug()
	} else {
		fmt.Printf("starting\n")
		v.Start()
		fmt.Printf("waiting to finish\n")
		err = v.Finish()
	}
	fmt.Printf("program finished %#v after %v instructions\n", err, v.Counter)
	v.SaveMetadata()
}
