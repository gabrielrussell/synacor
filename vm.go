package main

import (
	"encoding/binary"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ops = []func(*vM) error{
	opHalt,
	opSet,
	opPush,
	opPop,
	opEq,
	opGt,
	opJmp,
	opJT,
	opJF,
	opAdd,
	opMult,
	opMod,
	opAnd,
	opOr,
	opNot,
	opRMem,
	opWMem,
	opCall,
	opRet,
	opOut,
	opIn,
	opNoop,
}

type vM struct {
	Mem       []uint16
	Registers []uint16
	Stack     []uint16
	Ip        uint16
}

func (vm *vM) saveVM() error {
	file, err := os.Create(fmt.Sprintf("save-%v", time.Now().Format(time.RFC3339)))
	if err != nil {
		return err
	}
	encoder := gob.NewEncoder(file)
	return encoder.Encode(vm)
}

func (vm *vM) loadVM(fn string) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	decoder := gob.NewDecoder(file)
	return decoder.Decode(vm)

}

func (vm *vM) load(fn string) error {
	vm.Registers = make([]uint16, 8)
	vm.Stack = make([]uint16, 0, 64)
	vm.Ip = 0
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	fileinfo, err := file.Stat()
	if err != nil {
		return err
	}
	if fileinfo.Size()%2 == 1 {
		return fmt.Errorf("corrupt program file")
	}
	vm.Mem = make([]uint16, fileinfo.Size()/2)
	err = binary.Read(file, binary.LittleEndian, vm.Mem)
	if err != nil {
		return err
	}
	return nil
}

func (vm *vM) value(v uint16) uint16 {
	if v <= 32767 {
		return v
	}
	if v >= 32776 {
		panic("value out of range")
	}
	return vm.Registers[v-32768]
}

func (vm *vM) dest(d uint16) *uint16 {
	if d <= 32767 {
		return &vm.Mem[d]
	}
	if d >= 32776 {
		panic("dest out of range")
	}
	return &vm.Registers[d-32768]
}

func mod(v uint16) uint16 {
	return v % 32768
}

func opHalt(vm *vM) error {
	return fmt.Errorf("halt")
}

func opSet(vm *vM) error {
	a, b := vm.operand2()
	vm.Registers[a-32768] = vm.value(b)
	return nil
}
func opPush(vm *vM) error {
	a := vm.operand()
	vm.Stack = append(vm.Stack, vm.value(a))
	return nil
}
func opPop(vm *vM) error {
	a := vm.operand()
	*vm.dest(a) = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return nil
}
func opEq(vm *vM) error {
	a, b, c := vm.operand3()
	if vm.value(b) == vm.value(c) {
		*vm.dest(a) = 1
	} else {
		*vm.dest(a) = 0
	}
	return nil
}
func opGt(vm *vM) error {
	a, b, c := vm.operand3()
	if vm.value(b) > vm.value(c) {
		*vm.dest(a) = 1
	} else {
		*vm.dest(a) = 0
	}
	return nil
}
func opJmp(vm *vM) error {
	a := vm.operand()
	vm.Ip = vm.value(a)
	return nil
}
func opJT(vm *vM) error {
	a, b := vm.operand2()
	if vm.value(a) != 0 {
		vm.Ip = vm.value(b)
	}
	return nil
}
func opJF(vm *vM) error {
	a, b := vm.operand2()
	if vm.value(a) == 0 {
		vm.Ip = vm.value(b)
	}
	return nil
}
func opAdd(vm *vM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) + vm.value(c))
	return nil
}
func opMult(vm *vM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) * vm.value(c))
	return nil
}
func opMod(vm *vM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) % vm.value(c))
	return nil
}
func opAnd(vm *vM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = (vm.value(b) & vm.value(c)) & 0x7FFF
	return nil
}
func opOr(vm *vM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = (vm.value(b) | vm.value(c)) & 0x7FFF
	return nil
}
func opNot(vm *vM) error {
	a, b := vm.operand2()
	*vm.dest(a) = (^vm.value(b)) & 0x7FFF
	return nil
}
func opRMem(vm *vM) error {
	a, b := vm.operand2()
	*vm.dest(a) = vm.Mem[vm.value(b)]
	return nil
}
func opWMem(vm *vM) error {
	a, b := vm.operand2()
	vm.Mem[vm.value(a)] = vm.value(b)
	return nil
}
func opCall(vm *vM) error {
	a := vm.operand()
	vm.Stack = append(vm.Stack, vm.Ip)
	vm.Ip = vm.value(a)
	return nil
}
func opRet(vm *vM) error {
	if len(vm.Stack) == 0 {
		return fmt.Errorf("halt")
	}
	vm.Ip = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return nil
}
func opOut(vm *vM) error {
	a := vm.operand()
	c := vm.value(a)
	if c > 127 {
		return fmt.Errorf("bad ascii value %v", c)
	}
	fmt.Printf("%v", string([]byte{byte(c)}))
	return nil
}
func opIn(vm *vM) error {
	a := vm.operand()
	buf := make([]byte, 1)
	_, err := io.ReadFull(os.Stdin, buf)
	if err != nil {
		return err
	}
	//fmt.Printf("%v", string(buf))
	*vm.dest(a) = uint16(buf[0])
	return nil
}
func opNoop(vm *vM) error {
	return nil
}

func (vm *vM) operand() uint16 {
	v := vm.Mem[vm.Ip]
	vm.Ip++
	return v
}

func (vm *vM) operand2() (uint16, uint16) {
	return vm.operand(), vm.operand()
}
func (vm *vM) operand3() (uint16, uint16, uint16) {
	return vm.operand(), vm.operand(), vm.operand()
}

func (vm *vM) run() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1)
	for {
		op := vm.operand()
		err := ops[op](vm)
		if err != nil {
			if err.Error() == "halt" {
				return nil
			} else {
				return err
			}
		}
		select {
		case <-sigChan:
			vm.saveVM()
		default:
		}

	}
}

func main() {

	savedGame := flag.Bool("save", false, "load a saved vm instead of the program")
	flag.Parse()
	vm := &vM{}
	fmt.Fprintf(os.Stderr, "flags %v\n", flag.Args())
	if len(flag.Args()) != 1 {
		fmt.Printf("usage vm <program.bin>\n")
		os.Exit(1)
	}
	var err error
	if *savedGame {
		err = vm.loadVM(flag.Arg(0))
	} else {
		err = vm.load(flag.Arg(0))
	}
	if err != nil {
		fmt.Printf("load failed %v", err)
	}
	err = vm.run()
	if err != nil {
		fmt.Printf("program error %v", err)
		os.Exit(2)
	}
}
