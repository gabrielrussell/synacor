package vm

import (
	"fmt"
	"strconv"
)

func (vm *VM) Dis(p uint16, n int) []string {
	var dis []string
	for j := 0; j < n; j++ {
		s := fmt.Sprintf("%8d ", p)
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
		dis = append(dis, s)
	}
	return dis
}
