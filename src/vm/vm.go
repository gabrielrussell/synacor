package vm

import (
	"bufio"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Op struct {
	nFunction func(*VM, []*uint16) error
	name      string
	args      string
}

var Ops = []Op{
	{nOpHalt, opHalt, "Halt", ""},
	{nOpSet, opSet, "Set", "LR"},
	{nOpPush, opPush, "Push", "R"},
	{nOpPop, opPop, "Pop", "L"},
	{nOpEq, opEq, "Eq", "LRR"},
	{nOpGt, opGt, "Gt", "LRR"},
	{nOpJmp, opJmp, "Jmp", "R"},
	{nOpJT, opJT, "JT", "RR"},
	{nOpJF, opJF, "JF", "RR"},
	{nOpAdd, opAdd, "Add", "LRR"},
	{nOpMult, opMult, "Mult", "LRR"},
	{nOpMod, opMod, "Mod", "LRR"},
	{nOpAnd, opAnd, "And", "LRR"},
	{nOpOr, opOr, "Or", "LRR"},
	{nOpNot, opNot, "Not", "LR"},
	{nOpRMem, opRMem, "RMem", "LR"},
	{nOpWMem, opWMem, "WMem", "RR"},
	{nOpCall, opCall, "Call", "R"},
	{nOpRet, opRet, "Ret", ""},
	{nOpOut, opOut, "Out", "R"},
	{nOpIn, opIn, "In", "R"},
	{nOpNoop, opNoop, "Noop", ""},
}

type State struct {
	Mem       []uint16
	Registers []uint16
	Stack     []uint16
	Ip        uint16
}

type Metadata struct {
	Functions   map[uint16]bool
	ReadMem     []bool
	WriteMem    []bool
	ExecMem     []bool
	Annotations map[uint16]string
}

type VM struct {
	State
	Metadata
	MetadataFile string
	ControlChan  chan string
	SaveOnEOF    bool
	Breakpoints  map[uint16]bool
	Step         bool
	Stdout       io.Writer
	Stdin        *bufio.Reader
	Debugging    bool
	Counter      int
}

func (vm *VM) SaveMetadata() error {
	file, err := os.Create(vm.MetadataFile)
	if err != nil {
		return err
	}
	encoder := gob.NewEncoder(file)
	return encoder.Encode(vm.Metadata)
}

func (vm *VM) LoadMetadata() error {
	file, err := os.Open(vm.MetadataFile)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&vm.Metadata)
	}
	if len(vm.Metadata.ReadMem) < len(vm.Mem) {
		fmt.Printf("initilazing metadata\n")
		rMem := make([]bool, len(vm.Mem))
		copy(rMem, vm.ReadMem)
		vm.ReadMem = rMem
		wMem := make([]bool, len(vm.Mem))
		copy(wMem, vm.ReadMem)
		vm.WriteMem = wMem
		eMem := make([]bool, len(vm.Mem))
		copy(eMem, vm.ExecMem)
		vm.ExecMem = eMem
	}
	if vm.Metadata.Functions == nil {
		vm.Metadata.Functions = make(map[uint16]bool)
	}
	if vm.Metadata.Annotations == nil {
		vm.Metadata.Annotations = make(map[uint16]string)
	}
	return nil
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
		vm.WriteMem[d] = true
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

func nOpHalt(vm *VM, a []*uint16) error {
	return fmt.Errorf("halt")
}

func opHalt(vm *VM) error {
	return fmt.Errorf("halt")
}

func nOpSet(vm *VM, a []*uint16) error {
	*a[0] = *a[1]
	return nil
}

func opSet(vm *VM) error {
	a, b := vm.operand2()
	vm.Registers[a-32768] = vm.value(b)
	return nil
}
func nOpPush(vm *VM, a []*uint16) error {
	vm.Stack = append(vm.Stack, *a[0])
	return nil
}
func opPush(vm *VM) error {
	a := vm.operand()
	vm.Stack = append(vm.Stack, vm.value(a))
	return nil
}
func nOpPop(vm *VM, a []*uint16) error {
	*a[0] = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return nil
}
func opPop(vm *VM) error {
	a := vm.operand()
	*vm.dest(a) = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return nil
}

func nOpEq(vm *VM, a []*uint16) error {
	if *a[1] == *a[2] {
		*a[0] = 1
	} else {
		*a[0] = 0
	}
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

func nOpGt(vm *VM, a []*uint16) error {
	if *a[1] > *a[2] {
		*a[0] = 1
	} else {
		*a[0] = 0
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

func nOpJmp(vm *VM, a []*uint16) error {
	vm.Ip = *a[0]
	return nil
}
func opJmp(vm *VM) error {
	a := vm.operand()
	vm.Ip = vm.value(a)
	return nil
}
func nOpJT(vm *VM, a []*uint16) error {
	if *a[0] != 0 {
		vm.Ip = *a[1]
	}
	return nil
}
func opJT(vm *VM) error {
	a, b := vm.operand2()
	if vm.value(a) != 0 {
		vm.Ip = vm.value(b)
	}
	return nil
}

func nOpJF(vm *VM, a []*uint16) error {
	if *a[0] == 0 {
		vm.Ip = *a[1]
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

func nOpAdd(vm *VM, a []*uint16) error {
	*a[0] = mod(*a[1] + *a[2])
	return nil
}

func opAdd(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) + vm.value(c))
	return nil
}

func nOpMult(vm *VM, a []*uint16) error {
	*a[0] = mod(*a[1] * *a[2])
	return nil
}

func opMult(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) * vm.value(c))
	return nil
}

func nOpMod(vm *VM, a []*uint16) error {
	*a[0] = mod(*a[1] % *a[2])
	return nil
}

func opMod(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = mod(vm.value(b) % vm.value(c))
	return nil
}

func nOpAnd(vm *VM, a []*uint16) error {
	*a[0] = (*a[1] & *a[2]) & 0x7FFF
	return nil
}

func opAnd(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = (vm.value(b) & vm.value(c)) & 0x7FFF
	return nil
}

func nOpOr(vm *VM, a []*uint16) error {
	*a[0] = (*a[1] | *a[2]) & 0x7FFF
	return nil
}

func opOr(vm *VM) error {
	a, b, c := vm.operand3()
	*vm.dest(a) = (vm.value(b) | vm.value(c)) & 0x7FFF
	return nil
}
func nOpNot(vm *VM, a []*uint16) error {
	*a[0] = (^*a[1]) & 0x7FFF
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

func nOpRMem(vm *VM, a []*uint16) error {
	*a[0] = vm.Mem[*a[1]]
	return nil
}
func opWMem(vm *VM) error {
	a, b := vm.operand2()
	vm.Mem[vm.value(a)] = vm.value(b)
	return nil
}
func nOpWMem(vm *VM, a []*uint16) error {
	vm.WriteMem[*a[0]] = true
	vm.Mem[*a[0]] = *a[1]
	return nil
}

func opCall(vm *VM) error {
	a := vm.operand()
	vm.Stack = append(vm.Stack, vm.Ip)
	vm.Ip = vm.value(a)
	return nil
}

func nOpCall(vm *VM, a []*uint16) error {
	vm.Stack = append(vm.Stack, vm.Ip)
	vm.Ip = *a[0]
	return nil
}

func nOpRet(vm *VM, a []*uint16) error {
	if len(vm.Stack) == 0 {
		return fmt.Errorf("halt")
	}
	vm.Ip = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
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

func nOpOut(vm *VM, a []*uint16) error {
	fmt.Fprintf(vm.Stdout, "%v", string([]byte{byte(*a[0])}))
	return nil
}

func nOpIn(vm *VM, a []*uint16) error {
	buf := make([]byte, 1)
	_, err := io.ReadFull(vm.Stdin, buf)
	if err != nil {
		return err
	}
	*a[0] = uint16(buf[0])
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

func nOpNoop(vm *VM, a []*uint16) error {
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

func (vm *VM) Start() {
	fmt.Printf("Recieved state\n")
	<-vm.ControlChan
	fmt.Printf("signaling ready\n")
	vm.ControlChan <- ""
}

func (vm *VM) Finish() error {
	err := <-vm.ControlChan
	vm.ControlChan <- ""
	return fmt.Errorf("%v", err)
}

type decodedOp struct {
	Op
	Args            []*uint16
	ArgsDescription []string
}

func (vm *VM) Decode(p *uint16) (*decodedOp, error) {
	var dop decodedOp
	o := vm.Mem[*p]
	*p++
	if o > uint16(len(Ops)) {
		return nil, fmt.Errorf("op %v at %v is out of range", o, *p)
	}
	dop.Op = Ops[o]
	for _, arg := range dop.Op.args {
		var v *uint16
		var d string
		m := vm.Mem[*p]
		if m >= 32776 {
			return nil, fmt.Errorf("op value %v at %v out of range", m, *p)
		}
		if arg == 'R' {
			if m <= 32767 {
				v = &vm.Mem[*p]
				if dop.Op.name == "Out" {
					d = fmt.Sprintf("%c(*%v)", vm.Mem[*p], *p)
				} else {
					d = fmt.Sprintf("%v(*%v)", vm.Mem[*p], *p)
				}
			} else {
				v = &vm.Registers[m-32768]
				if dop.Op.name == "Out" {
					d = fmt.Sprintf("%c(R%v)", vm.Registers[m-32768], m-32768)
				} else {
					d = fmt.Sprintf("%v(R%v)", vm.Registers[m-32768], m-32768)
				}
			}
		} else {
			if m <= 32767 {
				v = &vm.Mem[m]
				d = fmt.Sprintf("*%v", *p)
			} else {
				v = &vm.Registers[m-32768]
				d = fmt.Sprintf("R%v", m-32768)
			}
		}
		dop.Args = append(dop.Args, v)
		dop.ArgsDescription = append(dop.ArgsDescription, d)
		*p++
	}
	return &dop, nil
}

func (vm *VM) Run() {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGUSR1)
	vm.Counter = 0
	vm.ControlChan <- "break"
	_, ok := <-vm.ControlChan
	if !ok {
		return
	}
	for {
		vm.Counter++
		opIp := vm.Ip
		decodeIp := vm.Ip
		var err error
		dOp, err := vm.Decode(&decodeIp)
		if err != nil {
			vm.ControlChan <- err.Error()
			<-vm.ControlChan
			return
		}
		vm.Ip = decodeIp
		err = dOp.nFunction(vm, dOp.Args)
		if err != nil {
			if err.Error() == "halt" {
				vm.ControlChan <- "halt"
				<-vm.ControlChan
				return
			} else if err == io.EOF {
				if vm.SaveOnEOF {
					vm.Ip = opIp
					vm.SaveVM("EOF")
				}
				vm.ControlChan <- "eof"
				<-vm.ControlChan
				return
			} else {
				vm.ControlChan <- err.Error()
				<-vm.ControlChan
				return
			}
		}
		if vm.Step || vm.Breakpoints[vm.Ip] {
			vm.ControlChan <- "break"
			_, ok := <-vm.ControlChan
			if !ok {
				return
			}
		}
	}
}

func (vm *VM) Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(vm.Stdout, format, a)
}
