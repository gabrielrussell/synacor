package vm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (vm *VM) Debug() error {
	for {
		state := <-vm.ControlChan
		if state != "break" {
			close(vm.ControlChan)
			return fmt.Errorf("%s", state)
		}
	replLoop:
		for {
			line, err := vm.Stdin.ReadString('\n')
			if err != nil {
				return err
			}
			fields := regexp.MustCompile(`\s`).Split(strings.TrimSpace(line), -1)
			switch fields[0] {
			case "s":
				vm.Step = true
				vm.Printf("stepping\n")
				break replLoop
			case "c":
				vm.Step = false
				vm.Printf("continuing\n")
				break replLoop
			case "d":
				p := vm.Ip
				l := 16
				var err error
				if len(fields) >= 3 {
					l, err = strconv.Atoi(fields[2])
				}
				if len(fields) >= 2 {
					var pint int
					pint, err = strconv.Atoi(fields[1])
					p = uint16(pint)
				}
				if err != nil {
					vm.Printf("d [<start> [<length>]\n")
					continue replLoop
				}
				vm.Printf("disassemble\n")
				dis := vm.Dis(p, l)
				for _, d := range dis {
					vm.Printf("%v\n", d)
				}
			default:
				vm.Printf("error, no such debugger command: %v\n", fields[0])
			}
		}
		vm.ControlChan <- ""
	}
}
