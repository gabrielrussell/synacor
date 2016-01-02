package vm

import (
	"fmt"
)

func (vm *VM) Dis(p uint16, n int) []string {
	var dis []string
	for j := 0; j < n; j++ {
		s := fmt.Sprintf("%8d ", p)
		dop, good := vm.Decode(&p, true)
		printChar := rune(dop.Code)
		if dop.Code < 32 || dop.Code > 127 {
			printChar = ' '
		}
		s += fmt.Sprintf("%2x %c ", dop.Code, printChar)
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
