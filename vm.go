package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

var ops = []func(*vM) error{
	opHalt,
	opSet,
	opPush,
	opPop,
	opEq,
	opGt,
	opJmp,
	opJt,
	opJf,
	opAdd,
	opMult,
	opMod,
	opAnd,
	opOr,
	opNot,
	opRmem,
	opWmem,
	opCall,
	opRet,
	opOut,
	opIn,
	opNoop,
}

type vM struct {
	mem       []uint16
	registers []uint16
	stack     []uint16
	ip        uint16
}

func (vm *vM) load(fn string) error {
	vm.registers = make([]uint16, 8)
	vm.stack = make([]uint16, 0, 64)
	vm.ip = 0
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
	vm.mem = make([]uint16, fileinfo.Size()/2)
	err = binary.Read(file, binary.LittleEndian, vm.mem)
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
	return vm.registers[v-32768]
}

func (vm *vM) dest(d uint16) *uint16 {
	if d <= 32767 {
		return &vm.mem[d]
	}
	if d >= 32776 {
		panic("dest out of range")
	}
	return &vm.registers[d-32768]
}

func mod(v uint16) uint16 {
	return v % 32768
}

func opHalt(vm *vM) error {
	return fmt.Errorf("halt")
}

func opSet(vm *vM) error {
	a, b := vm.operand2()
	vm.registers[a-32768] = vm.value(b)
	return nil
}
func opPush(vm *vM) error {
	a := vm.operand()
	vm.stack = append(vm.stack, vm.value(a))
	return nil
}
func opPop(vm *vM) error {
	a := vm.operand()
	*vm.dest(a) = vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
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
	vm.ip = vm.value(a)
	return nil
}
func opJt(vm *vM) error {
	a, b := vm.operand2()
	if vm.value(a) != 0 {
		vm.ip = vm.value(b)
	}
	return nil
}
func opJf(vm *vM) error {
	a, b := vm.operand2()
	if vm.value(a) == 0 {
		vm.ip = vm.value(b)
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
func opRmem(vm *vM) error {
	a, b := vm.operand2()
	*vm.dest(a) = vm.mem[vm.value(b)]
	return nil
}
func opWmem(vm *vM) error {
	a, b := vm.operand2()
	vm.mem[vm.value(a)] = vm.value(b)
	return nil
}
func opCall(vm *vM) error {
	a := vm.operand()
	vm.stack = append(vm.stack, vm.ip)
	vm.ip = vm.value(a)
	return nil
}
func opRet(vm *vM) error {
	if len(vm.stack) == 0 {
		return fmt.Errorf("halt")
	}
	vm.ip = vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
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
	fmt.Printf("%v", string(buf))
	*vm.dest(a) = uint16(buf[0])
	return nil
}
func opNoop(vm *vM) error {
	return nil
}

func (vm *vM) operand() uint16 {
	v := vm.mem[vm.ip]
	vm.ip++
	return v
}

func (vm *vM) operand2() (uint16, uint16) {
	return vm.operand(), vm.operand()
}
func (vm *vM) operand3() (uint16, uint16, uint16) {
	return vm.operand(), vm.operand(), vm.operand()
}

func (vm *vM) run() error {
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
	}
}

func main() {
	vm := &vM{}
	if len(os.Args) != 2 {
		fmt.Printf("usage vm <program.bin>\n")
		os.Exit(1)
	}
	err := vm.load(os.Args[1])
	if err != nil {
		fmt.Printf("load failed %v", err)
	}
	err = vm.run()
	if err != nil {
		fmt.Printf("program error %v", err)
		os.Exit(2)
	}
}
