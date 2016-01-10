package vm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (vm *VM) R() string {
	s := []string{}
	for i, r := range vm.Registers {
		s = append(s, fmt.Sprintf("R%v: %5v", i, strconv.Itoa(int(r))))
	}
	s = append(s, fmt.Sprintf("IP: %5v", strconv.Itoa(int(vm.Ip))))
	if len(vm.Stack) > 0 {
		s = append(s, fmt.Sprintf("S%v: %5v", len(vm.Stack)-1, strconv.Itoa(int(vm.Stack[len(vm.Stack)-1]))))
	}
	return strings.Join(s, ", ")
}

func (vm *VM) Debug() error {
	var fields []string
	var repeat int
	var lastLine string
	for {
		state := <-vm.ControlChan
		if state != "break" {
			close(vm.ControlChan)
			return fmt.Errorf("%s", state)
		}
		dis := vm.Dis(vm.Ip, 1)
		vm.Printf("%v\n%v\n", dis[0], vm.R())
	replLoop:
		for {
			vm.Printf("DBG> ")
			line, err := vm.Stdin.ReadString('\n')
			if err != nil {
				return err
			}
			if len(line) == 1 && len(lastLine) > 0 {
				repeat++
			} else {
				repeat = 0
				fields = regexp.MustCompile(`\s`).Split(strings.TrimSpace(line), -1)
				lastLine = line
			}
			switch fields[0] {
			case "s":
				vm.Step = true
				break replLoop
			case "c":
				vm.Step = false
				break replLoop
			case "d":
				p := vm.Ip
				l := 32
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
				p += uint16(l * repeat)
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
				vm.meta.Annotations[uint16(p)] = strings.Join(fields[2:], " ")
				dis := vm.Dis(uint16(p), 1)
				vm.Printf("%v\n", dis[0])
			case "bt":
				for i := 0; i < len(vm.CallStack); i += 2 {
					vm.Printf("call %v, from %v\n", vm.CallStack[i], vm.CallStack[i+1])

				}
			case "save":
				if len(fields) != 2 {
					vm.Printf("save <filename>\n")
					continue replLoop
				}
				err := vm.SaveVM(fields[1])
				if err != nil {
					vm.Printf("%v\n", err)
				}
			case "r":
				if len(fields) != 3 {
					vm.Printf("r <number> <value>\n")
					continue replLoop
				}
				r, err := strconv.Atoi(fields[1])
				if err != nil || r < 0 || r > 7 {
					vm.Printf("r <0-7> <value>\n")
					continue replLoop
				}
				v, err := strconv.Atoi(fields[2])
				if err != nil {
					vm.Printf("r <number> <value>\n")
					continue replLoop
				}
				vm.Registers[r] = uint16(v)
				dis := vm.Dis(vm.Ip, 1)
				vm.Printf("%v\n%v\n", dis[0], vm.R())
			case "m":
				usage := "m <address> <value>\n"
				if len(fields) != 3 {
					vm.Printf(usage)
					continue replLoop
				}
				p, err := strconv.Atoi(fields[1])
				if err != nil || p > len(vm.Mem) {
					vm.Printf(usage)
					continue replLoop
				}
				v, err := strconv.Atoi(fields[2])
				if err != nil {
					vm.Printf(usage)
					continue replLoop
				}
				vm.Mem[p] = uint16(v)
			case "string":
				if len(fields) != 2 {
					vm.Printf("string <address>\n")
				}
				p, err := strconv.Atoi(fields[1])
				if err != nil {
					vm.Printf("string <address>\n")
					continue replLoop
				}
				vm.Printf("%v\n", vm.String(uint16(p)))
			case "l", "look":
				if len(fields) != 2 {
					vm.Printf("l <address>\n")
					continue replLoop
				}
				pInt, err := strconv.Atoi(fields[1])
				if err != nil {
					vm.Printf("l <address>\n")
					continue replLoop
				}
				p := uint16(pInt)
				found := 0
				for i := uint16(0); i < uint16(len(vm.Mem)); i++ {
					if vm.Mem[i] == p {
						vm.Printf("%v\n", i)
						found++
						if found > 100 {
							break
						}
					}

				}
			case "binary", "bin":
				if len(fields) != 2 {
					vm.Printf("bin[ary] <number>\n")
					continue replLoop
				}
				n, err := strconv.Atoi(fields[1])
				if err != nil {
					vm.Printf("bin[ary] <number>\n")
					continue replLoop
				}
				vm.Printf("%016b\n", n)
			default:
				vm.Printf("error, no such debugger command: %v\n", fields[0])
			}
		}
		vm.ControlChan <- ""
	}
}
