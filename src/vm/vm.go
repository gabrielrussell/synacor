package vm

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ops = []func(*VM) error{
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

type State struct {
	Mem       []uint16
	Registers []uint16
	Stack     []uint16
	Ip        uint16
}

type VM struct {
	State
	Step        bool
	StepChan    chan interface{}
	SaveOnEOF   bool
	breakpoints []int16
	Stdout      io.Writer
	Stdin       io.Reader
}

func (vm *VM) SaveVM(name string) error {
	file, err := os.Create(fmt.Sprintf("%v-%v", name, time.Now().Format(time.RFC3339)))
	if err != nil {
		return err
	}
	encoder := gob.NewEncoder(file)
	return encoder.Encode(vm.State)
}

func (vm *VM) LoadVM(fn string) error {
	file, err := os.Open(fn)
	if err != nil {
		return fmt.Errorf("%v \"%v\"", err, fn)
	}
	decoder := gob.NewDecoder(file)
	return decoder.Decode(&vm.State)

}

func (vm *VM) Load(fn string) error {
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

func (vm *VM) value(v uint16) uint16 {
	if v <= 32767 {
		return v
	}
	if v >= 32776 {
		panic("value out of range")
	}
	return vm.Registers[v-32768]
}

func (vm *VM) dest(d uint16) *uint16 {
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

func opHalt(vm *VM) error {
	return fmt.Errorf("halt")
}

func opSet(vm *VM) error {
	a, b := vm.operand2()
	vm.Registers[a-32768] = vm.value(b)
	return nil
}
func opPush(vm *VM) error {
	a := vm.operand()
	vm.Stack = append(vm.Stack, vm.value(a))
	return nil
}
func opPop(vm *VM) error {
	a := vm.operand()
	*vm.dest(a) = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return nil
}
func opEq(vm *VM) error {
	a, b, c := vm.operand3()
	if vm.value(b) == vm.value(c) {
		*vm.dest(a) = 1
	} else {
		*vm.dest(a) = 0
	}
	return nil
}
func opGt(vm *VM) error {
	a, b, c := vm.operand3()
	if vm.value(b) > vm.value(c) {
		*vm.dest(a) = 1
	} else {
		*vm.dest(a) = 0
	}
	return nil
}
func opJmp(vm *VM) error {
	a := vm.operand()
	vm.Ip = vm.value(a)
	return nil
}
func opJT(vm *VM) error {
	a, b := vm.operand2()
	if vm.value(a) != 0 {
		vm.Ip = vm.value(b)
	}
	return nil
}
func opJF(vm *VM) error {
	a, b := vm.operand2()
	if vm.value(a) == 0 {
		vm.Ip = vm.value(b)
	}
	return nil
}
func opAdd(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) + vm.value(c))
	return nil
}
func opMult(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) * vm.value(c))
	return nil
}
func opMod(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) % vm.value(c))
	return nil
}
func opAnd(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = (vm.value(b) & vm.value(c)) & 0x7FFF
	return nil
}
func opOr(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = (vm.value(b) | vm.value(c)) & 0x7FFF
	return nil
}
func opNot(vm *VM) error {
	a, b := vm.operand2()
	*vm.dest(a) = (^vm.value(b)) & 0x7FFF
	return nil
}
func opRMem(vm *VM) error {
	a, b := vm.operand2()
	*vm.dest(a) = vm.Mem[vm.value(b)]
	return nil
}
func opWMem(vm *VM) error {
	a, b := vm.operand2()
	vm.Mem[vm.value(a)] = vm.value(b)
	return nil
}
func opCall(vm *VM) error {
	a := vm.operand()
	vm.Stack = append(vm.Stack, vm.Ip)
	vm.Ip = vm.value(a)
	return nil
}
func opRet(vm *VM) error {
	if len(vm.Stack) == 0 {
		return fmt.Errorf("halt")
	}
	vm.Ip = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return nil
}
func opOut(vm *VM) error {
	a := vm.operand()
	c := vm.value(a)
	if c > 127 {
		return fmt.Errorf("bad ascii value %v", c)
	}
	fmt.Fprintf(vm.Stdout, "%v", string([]byte{byte(c)}))
	return nil
}
func opIn(vm *VM) error {
	a := vm.operand()
	buf := make([]byte, 1)
	_, err := io.ReadFull(vm.Stdin, buf)
	if err != nil {
		return err
	}
	//fmt.Printf("%v", string(buf))
	*vm.dest(a) = uint16(buf[0])
	return nil
}
func opNoop(vm *VM) error {
	return nil
}

func (vm *VM) operand() uint16 {
	v := vm.Mem[vm.Ip]
	vm.Ip++
	return v
}

func (vm *VM) operand2() (uint16, uint16) {
	return vm.operand(), vm.operand()
}
func (vm *VM) operand3() (uint16, uint16, uint16) {
	return vm.operand(), vm.operand(), vm.operand()
}

func (vm *VM) Run() error {
	signal.Notify(sigChan, syscall.SIGUSR1)
	for {
		opIp := vm.Ip
		if vm.Step {
			<-vm.StepChan
		}
		op := vm.operand()
		err := ops[op](vm)
		if err != nil {
			if err.Error() == "halt" {
				return nil
			} else if err == io.EOF {
				if vm.SaveOnEOF {
					vm.Ip = opIp
					vm.SaveVM("EOF")
				}
			} else {
				return err
			}
		}
		select {
		case <-sigChan:
			vm.SaveVM("autosave")
		default:
		}

	}
}
