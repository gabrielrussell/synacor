package vm

import (
	"fmt"
)

func (vm *VM) Dis(p uint16, n int) []string {
	var dis []string
	for j := 0; j < n; j++ {
		s := fmt.Sprintf("%8d ", p)
		dop, _ := vm.Decode(&p)
		s += dop.name
		for k := 0; k < len(dop.ArgsDescription); k++ {
			s += ", " + dop.ArgsDescription[k]
		}
		dis = append(dis, s)
	}
	return dis
}
