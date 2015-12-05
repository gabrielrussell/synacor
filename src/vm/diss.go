package vm

import (
	"fmt"
)

var opsArgs = [][]string{
	{"Halt", ""},
	{"Set", "LR"},
	{"Push", "R"},
	{"Pop", "L"},
	{"Eq", "LRR"},
	{"Gt", "LRR"},
	{"Jmp", "R"},
	{"JT", "RR"},
	{"JF", "RR"},
	{"Add", "LRR"},
	{"Mult", "LRR"},
	{"Mod", "LRR"},
	{"And", "LRR"},
	{"Or", "LRR"},
	{"Not", "LR"},
	{"RMem", "LR"},
	{"WMem", "LR"},
	{"Call", "R"},
	{"Ret", ""},
	{"Out", "R"},
	{"In", "R"},
	{"Noop", ""},
}

func RValue(v uint16) string {
	if v <= 32767 {
		return fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf("*R%v", v-32768)
}

func LValue(d uint16) string {
	if d <= 32767 {
		return fmt.Sprintf("*%v", d)
	}
	return fmt.Sprintf("R%v", d-32768)
}

func (vm *VM) Disassemble(count uint16) [][]string {
	dis := [][]string{}
	i := vm.Ip
	for i < vm.Ip+count {
		opDis := []string{fmt.Sprintf("%v", i)}
		op := vm.Mem[i]
		i++
		if op >= uint16(len(opsArgs)) {
			fmt.Printf("op: %v\n", op)
			continue
		}
		opName := opsArgs[op][0]
		opDis = append(opDis, opName)
		opArgs := opsArgs[op][1]
		for j := 0; j < len(opArgs); j++ {
			if op == 19 {
				opDis = append(opDis, fmt.Sprintf("\"%v\"", string([]byte{byte(vm.Mem[i])})))
			} else {
				if opArgs[j] == 'R' {
					opDis = append(opDis, RValue(vm.Mem[i]))
				} else {
					opDis = append(opDis, LValue(vm.Mem[i]))
				}
			}
			i++
		}
		fmt.Printf("%v\n", opDis)
		dis = append(dis, opDis)
	}
	return dis
}
