package main

import (
	"fmt"
	"strconv"
)

type walkState struct {
	x     int
	y     int
	sum   uint16
	op    string
	math  string
	depth int
	path  string
	maze  [][]string
}

func (s *walkState) String() string {
	return fmt.Sprintf("x:%v y:%v sum:%v math:%v op:%v depth:%v path:%v", s.x, s.y, s.sum, s.math, s.op, s.depth, s.path)
}

func (state *walkState) calc() {
	state.depth++
	state.math = state.math + state.here()
	if state.op != "" {
		vInt, _ := strconv.Atoi(state.here())
		v := uint16(vInt)
		switch state.op {
		case "+":
			state.sum += v
		case "-":
			state.sum -= v
		case "*":
			state.sum *= v
		}
		state.sum = state.sum % 32768
	}
	fmt.Printf("%v\n", state)
}

func (state *walkState) here() string {
	return state.maze[state.x][state.y]
}

func (state *walkState) genWalk() []*walkState {
	var newStates []*walkState
	var newOp string
	if state.op == "" {
		newOp = state.here()
	} else {
		newOp = ""
	}
	if state.x == 3 && state.y == 3 {
		return newStates //empty
	}
	if state.x == 0 && state.y == 0 {
		return newStates //empty
	}
	if state.x > 0 {
		ns := *state
		ns.x--
		ns.path += "s"
		ns.op = newOp
		newStates = append(newStates, &ns)
	}
	if state.x < 3 {
		ns := *state
		ns.x++
		ns.path += "n"
		ns.op = newOp
		newStates = append(newStates, &ns)
	}
	if state.y > 0 {
		ns := *state
		ns.y--
		ns.path += "w"
		ns.op = newOp
		newStates = append(newStates, &ns)
	}
	if state.y < 3 {
		ns := *state
		ns.y++
		ns.path += "e"
		ns.op = newOp
		newStates = append(newStates, &ns)
	}
	return newStates
}

func (state *walkState) isSuccess() bool {
	return state.x == 3 && state.y == 3 && state.sum == 30
}
func (state *walkState) printSolution() {
	m := map[rune]string{
		'n': "north",
		's': "south",
		'e': "east",
		'w': "west",
	}
	for _, c := range state.path {
		fmt.Printf("%v\n", m[c])
	}
}

func main() {
	maze := [][]string{
		{"22", "-", "9", "*"},
		{"+", "4", "-", "18"},
		{"4", "*", "11", "*"},
		{"*", "8", "-", "1"},
	}
	queue := []*walkState{
		&walkState{
			x:    1,
			y:    0,
			sum:  22,
			op:   "",
			math: "22",
			path: "n",
			maze: maze},
		&walkState{
			x:    0,
			y:    1,
			sum:  22,
			op:   "",
			math: "22",
			path: "e",
			maze: maze}}
	for {
		state := queue[0]
		queue = queue[1:]
		state.calc()
		if state.isSuccess() {
			state.printSolution()
			break
		}
		queue = append(queue, state.genWalk()...)
	}
}
