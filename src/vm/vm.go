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
	Function func(*VM, []*uint16) error
	Name     string
	Args     string
}

var Ops = []Op{
	{OpHalt, "Halt", ""},
	{OpSet, "Set", "LR"},
	{OpPush, "Push", "R"},
	{OpPop, "Pop", "L"},
	{OpEq, "Eq", "LRR"},
	{OpGt, "Gt", "LRR"},
	{OpJmp, "Jmp", "R"},
	{OpJT, "JT", "RR"},
	{OpJF, "JF", "RR"},
	{OpAdd, "Add", "LRR"},
	{OpMult, "Mult", "LRR"},
	{OpMod, "Mod", "LRR"},
	{OpAnd, "And", "LRR"},
	{OpOr, "Or", "LRR"},
	{OpNot, "Not", "LR"},
	{OpRMem, "RMem", "LR"},
	{OpWMem, "WMem", "RR"},
	{OpCall, "Call", "R"},
	{OpRet, "Ret", ""},
	{OpOut, "Out", "R"},
	{OpIn, "In", "R"},
	{OpNoop, "Noop", ""},
}

type State struct {
	Mem       []uint16
	Registers []uint16
	Stack     []uint16
	CallStack []uint16
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
	meta         Metadata
	MetadataFile string
	ControlChan  chan string
	SaveOnEOF    bool
	Break        map[uint16]bool
	BreakOps     map[uint16]bool
	Step         bool
	Stdout       io.Writer
	Stdin        *bufio.Reader
	Debugging    bool
	Counter      int
}

func (vm *VM) SaveMetadata() error {
	vm.Printf("saving metadata\n")
	file, err := os.Create(vm.MetadataFile)
	if err != nil {
		return err
	}
	encoder := gob.NewEncoder(file)
	return encoder.Encode(vm.meta)
}

func (vm *VM) LoadMetadata() error {
	file, err := os.Open(vm.MetadataFile)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&vm.meta)
	}
	if len(vm.meta.ReadMem) < len(vm.Mem) {
		fmt.Printf("initilazing metadata\n")
		rMem := make([]bool, len(vm.Mem))
		copy(rMem, vm.meta.ReadMem)
		vm.meta.ReadMem = rMem
		wMem := make([]bool, len(vm.Mem))
		copy(wMem, vm.meta.ReadMem)
		vm.meta.WriteMem = wMem
		eMem := make([]bool, len(vm.Mem))
		copy(eMem, vm.meta.ExecMem)
		vm.meta.ExecMem = eMem
	}
	if vm.meta.Functions == nil {
		vm.meta.Functions = make(map[uint16]bool)
	}
	if vm.meta.Annotations == nil {
		vm.meta.Annotations = make(map[uint16]string)
	}
	return nil
}

func (vm *VM) SaveVM(name string) error {
	fn := fmt.Sprintf("%v-%v", name, time.Now().Format(time.RFC3339))
	file, err := os.Create(fn)
	if err != nil {
		return err
	}
	vm.Printf("saving to %v\n", fn)
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
		vm.meta.WriteMem[d] = true
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

func OpHalt(vm *VM, a []*uint16) error {
	return fmt.Errorf("halt")
}

func OpSet(vm *VM, a []*uint16) error {
	*a[0] = *a[1]
	return nil
}

func OpPush(vm *VM, a []*uint16) error {
	vm.Stack = append(vm.Stack, *a[0])
	return nil
}
func OpPop(vm *VM, a []*uint16) error {
	*a[0] = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return nil
}

func OpEq(vm *VM, a []*uint16) error {
	if *a[1] == *a[2] {
		*a[0] = 1
	} else {
		*a[0] = 0
	}
	return nil
}

func OpGt(vm *VM, a []*uint16) error {
	if *a[1] > *a[2] {
		*a[0] = 1
	} else {
		*a[0] = 0
	}
	return nil
}

func OpJmp(vm *VM, a []*uint16) error {
	vm.Ip = *a[0]
	return nil
}

func OpJT(vm *VM, a []*uint16) error {
	if *a[0] != 0 {
		vm.Ip = *a[1]
	}
	return nil
}

func OpJF(vm *VM, a []*uint16) error {
	if *a[0] == 0 {
		vm.Ip = *a[1]
	}
	return nil
}

func OpAdd(vm *VM, a []*uint16) error {
	*a[0] = mod(*a[1] + *a[2])
	return nil
}

func OpMult(vm *VM, a []*uint16) error {
	*a[0] = mod(*a[1] * *a[2])
	return nil
}

func OpMod(vm *VM, a []*uint16) error {
	*a[0] = mod(*a[1] % *a[2])
	return nil
}

func OpAnd(vm *VM, a []*uint16) error {
	*a[0] = (*a[1] & *a[2]) & 0x7FFF
	return nil
}

func OpOr(vm *VM, a []*uint16) error {
	*a[0] = (*a[1] | *a[2]) & 0x7FFF
	return nil
}

func OpNot(vm *VM, a []*uint16) error {
	*a[0] = (^*a[1]) & 0x7FFF
	return nil
}

func OpRMem(vm *VM, a []*uint16) error {
	*a[0] = vm.Mem[*a[1]]
	return nil
}

func OpWMem(vm *VM, a []*uint16) error {
	vm.meta.WriteMem[*a[0]] = true
	vm.Mem[*a[0]] = *a[1]
	return nil
}

func OpCall(vm *VM, a []*uint16) error {
	vm.meta.Functions[*a[0]] = true
	vm.CallStack = append(vm.CallStack, *a[0], vm.Ip-2)
	vm.Stack = append(vm.Stack, vm.Ip)
	vm.Ip = *a[0]
	return nil
}

func OpRet(vm *VM, a []*uint16) error {
	if len(vm.Stack) == 0 {
		return fmt.Errorf("halt")
	}
	vm.Ip = vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	if len(vm.CallStack) > 0 {
		vm.CallStack = vm.CallStack[:len(vm.CallStack)-2]
	}
	return nil
}

func OpOut(vm *VM, a []*uint16) error {
	fmt.Fprintf(vm.Stdout, "%v", string([]byte{byte(*a[0])}))
	return nil
}

func OpIn(vm *VM, a []*uint16) error {
	buf := make([]byte, 1)
	_, err := io.ReadFull(vm.Stdin, buf)
	if err != nil {
		return err
	}
	*a[0] = uint16(buf[0])
	return nil
}

func OpNoop(vm *VM, a []*uint16) error {
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
	vm.Printf("Recieved state\n")
	<-vm.ControlChan
	vm.Printf("signaling ready\n")
	vm.ControlChan <- ""
}

func (vm *VM) Finish() error {
	err := <-vm.ControlChan
	vm.ControlChan <- ""
	return fmt.Errorf("%v", err)
}

type decodedOp struct {
	Op
	Codes           []uint16
	Args            []*uint16
	ArgsDescription []string
	Annotation      string
	isFunction      bool
}

func (vm *VM) Decode(p *uint16, verbose bool) (*decodedOp, bool) {
	var dop decodedOp
	if int(*p) >= len(vm.Mem) {
		return &dop, false
	}
	//start := *p
	o := vm.Mem[*p]
	dop.Annotation = vm.meta.Annotations[*p]
	dop.isFunction = vm.meta.Functions[*p]
	dop.Codes = append(dop.Codes, o)
	*p++
	if o >= uint16(len(Ops)) {
		return &dop, false
	}
	dop.Op = Ops[o]
	for _, arg := range dop.Op.Args {
		var v *uint16
		var d string
		m := &vm.Mem[*p]
		dop.Codes = append(dop.Codes, *m)
		if *m >= 32776 {
			return &dop, false
		}
		if arg == 'R' {
			if *m <= 32767 {
				v = m
				if verbose {
					if dop.Op.Name == "Out" {
						d = fmt.Sprintf("%c", *v)
					} else {
						d = fmt.Sprintf("%v", *v)
					}
				}
			} else {
				v = &vm.Registers[*m-32768]
				if verbose {
					if dop.Op.Name == "Out" {
						d = fmt.Sprintf("%c(R%v)", vm.Registers[*m-32768], *m-32768)
					} else {
						d = fmt.Sprintf("%v(R%v)", vm.Registers[*m-32768], *m-32768)
					}
				}
			}
		} else {
			if *m <= 32767 {
				v = &vm.Mem[*m]
				if verbose {
					d = fmt.Sprintf("*%v", *m)
				}
			} else {
				v = &vm.Registers[*m-32768]
				if verbose {
					d = fmt.Sprintf("R%v", *m-32768)
				}
			}
		}
		if dop.Codes[0] == 17 { // decorate the parameter to call with any annotations on the address
			if vm.meta.Annotations[*v] != "" {
				d += "(" + vm.meta.Annotations[*v] + ")"
			}
		}
		dop.Args = append(dop.Args, v)
		if verbose {
			dop.ArgsDescription = append(dop.ArgsDescription, d)
		}
		*p++
	}
	return &dop, true
}

func (vm *VM) Run() {

	saveSigChan := make(chan os.Signal, 1)
	signal.Notify(saveSigChan, syscall.SIGUSR1)

	dbgSigChan := make(chan os.Signal, 1)
	if vm.Debugging {

		signal.Notify(dbgSigChan, syscall.SIGINT)
	}
	vm.Counter = 0
	vm.ControlChan <- "break"
	_, ok := <-vm.ControlChan
	if !ok {
		return
	}
	for {
		vm.Counter++
		opIp := vm.Ip
		var err error
		var receivedDbgSig bool
		select {
		case <-saveSigChan:
			vm.SaveVM("SIG")
		case sig := <-dbgSigChan:
			vm.Printf("SIGNAL RECEIVED %v\n", sig)
			receivedDbgSig = true
		default:
		}
		if vm.BreakOps[vm.Mem[vm.Ip]] || vm.Step || vm.Break[vm.Ip] || receivedDbgSig {
			vm.ControlChan <- "break"
			_, ok := <-vm.ControlChan
			if !ok {
				return
			}
		}
		vm.meta.ExecMem[vm.Ip] = true
		dOp, good := vm.Decode(&vm.Ip, false)
		if !good {
			vm.ControlChan <- fmt.Sprintf("bad op %v at %v", dOp.Codes[0], opIp)
			<-vm.ControlChan
			return
		}
		err = dOp.Function(vm, dOp.Args)
		if err != nil {
			if err.Error() == "halt" {
				vm.ControlChan <- "halt"
				<-vm.ControlChan
				return
			} else if err == io.EOF {
				if vm.Debugging {
					vm.Ip = opIp
					vm.Step = true
					continue
				}
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
	}
}

func (vm *VM) Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(vm.Stdout, format, a...)
}
