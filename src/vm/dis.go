package vm

import (
	"fmt"
	"strconv"
)

func (vm *VM) Dis(p uint16, n int) []string {
	var dis []string
	start := p
	for p < start+uint16(n) {
		s := fmt.Sprintf("%8d ", p)
		str := vm.String(p)
		if str == "" && vm.Mem[p] < uint16(len(vm.Mem)) && vm.Mem[p] > uint16(512) {
			str = vm.String(vm.Mem[p])
		}
		dop, good := vm.Decode(&p, true)
		var chars [4]byte
		var values string
		for i := 0; i < 4; i++ {
			if i < len(dop.Codes) {
				if len(values) > 0 {
					values += ","
				}
				values += strconv.Itoa(int(dop.Codes[i]))
			}
			if i >= len(dop.Codes) || dop.Codes[i] < 32 || dop.Codes[i] > 127 {
				chars[i] = ' '
			} else {
				chars[i] = byte(dop.Codes[i])

			}
		}
		s += fmt.Sprintf("%25v '%v' ", values, string(chars[:]))
		if good {
			if dop.isFunction {
				s += "* "
			} else {
				s += "  "
			}
			s += dop.Name
			for k := 0; k < len(dop.ArgsDescription); k++ {
				s += ", " + dop.ArgsDescription[k]
			}
			if dop.Annotation != "" {
				s += " # " + dop.Annotation
			}
		}
		if str != "" {
			s += " " + str
		}
		dis = append(dis, s)
	}
	return dis
}

func (vm *VM) String(p uint16) string {
	len := vm.Mem[p]
	if len > 1024 || len == 0 {
		return ""
	}
	var b [1024]byte
	for i := uint16(0); i < len; i++ {
		if (vm.Mem[p+i+1] < 32 && vm.Mem[p+i+1] != 10) || vm.Mem[p+i+1] > 128 {
			return ""
		} else {
			b[i] = byte(vm.Mem[p+i+1])
		}
	}
	return fmt.Sprintf("%v:%v:\"%v\"", p, len, string(b[:len]))
}
