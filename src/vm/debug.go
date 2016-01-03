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
		dis := vm.Dis(vm.Ip, 1)
		vm.Printf("%v\n", dis[0])

	replLoop:
		for {
			vm.Printf("DBG> ")
			line, err := vm.Stdin.ReadString('\n')
			if err != nil {
				return err
			}
			fields := regexp.MustCompile(`\s`).Split(strings.TrimSpace(line), -1)
			switch fields[0] {
			case "s":
				vm.Step = true
				break replLoop
			case "c":
				vm.Step = false
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
			case "op":
				if len(fields) > 2 {
					vm.Printf("op [<opName>]\n")
					continue replLoop
				}
				if len(fields) == 1 {
					vm.BreakOps = make(map[uint16]bool)
					continue replLoop
				}
				var found bool
				var op uint16
				for o := range Ops {
					if Ops[o].Name == fields[1] {
						found = true
						op = uint16(o)
					}
				}
				if !found {
					vm.Printf("op [<opName>]\n")
					continue replLoop
				}
				vm.BreakOps[op] = true
			case "break", "b":
				if len(fields) != 2 {
					vm.Printf("break <addr>\n")
					continue replLoop
				}
				l, err := strconv.Atoi(fields[1])
				if err != nil {
					vm.Printf("break <addr>\n%v\n", err)
					continue replLoop
				}
				vm.Break[uint16(l)] = true
			case "del":
				if len(fields) != 2 {
					vm.Printf("del <addr>\n")
					continue replLoop
				}
				l, err := strconv.Atoi(fields[1])
				if err != nil {
					vm.Printf("del <addr>\n%v\n", err)
					continue replLoop
				}
				vm.Break[uint16(l)] = false
			case "ann":
				if len(fields) < 3 {
					vm.Printf("ann <addr> <note>\n")
					continue replLoop
				}
				p, err := strconv.Atoi(fields[1])
				if err != nil {
					vm.Printf("ann <addr> <note>\n%v\n", err)
					continue replLoop
				}
				vm.Annotations[uint16(p)] = strings.Join(fields[2:], " ")
				dis := vm.Dis(uint16(p), 1)
				vm.Printf("%v\n", dis[0])
			case "bt":
				for i := 0; i < len(vm.CallStack); i += 2 {
					vm.Printf("call %v, from %v\n", vm.CallStack[i], vm.CallStack[i+1])

				}
			default:
				vm.Printf("error, no such debugger command: %v\n", fields[0])
			}
		}
		vm.ControlChan <- ""
	}
}
