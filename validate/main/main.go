package main

import (
	"fmt"
)

const fifteenbit uint16 = 32768

func add(a uint16, b uint16) uint16 {
	return (a + b) % fifteenbit
}

var r7 uint16

var cache map[uint32]uint32

func key(a uint16, b uint16) uint32 {
	return uint32(a)*65536 + uint32(b)
}
func store(a uint16, b uint16, c uint16, d uint16) {
	cache[key(a, b)] = key(c, d)
}

// 6027
func validate(r0 uint16, r1 uint16) (uint16, uint16) {
	if val, ok := cache[key(r0, r1)]; ok {
		r1 = uint16(val & (65536 - 1))
		r0 = uint16(val / 65536)
		return r0, r1
	}
	saveR0 := r0
	saveR1 := r1
	// JT, (R0), 6035
	if r0 == 0 {
		// Add, R0, (R1), 1
		// Ret
		r0 = r1 + 1
		r1 = r1
		store(saveR0, saveR1, r0, r1)
		return r0, r1
	}
	// 6025
	// JT, (R1), 6048
	if r1 == 0 {
		fmt.Printf("-")
		// Add, R0, (R0), 32767
		// Set, R1, (R7)
		// Call, 6027
		// Ret
		r0, r1 = validate(add(r0, 32767), r7)
		store(saveR0, saveR1, r0, r1)
		return r0, r1
	}
	// 6048
	// Push, (R0)
	// Add, R1, (R1), 32767
	// Call, 6027
	// Set, R1, (R0)
	// Pop, R0
	// Add, R0, (R0), 32767
	// Call, 6027
	r1, _ = validate(r0, add(r1, 32767))
	r0, r1 = validate(add(r0, 32767), r1)
	store(saveR0, saveR1, r0, r1)
	return r0, r1
}

func main() {
	for r7 = 0; r7 < 32767; r7++ {
		cache = make(map[uint32]uint32)
		r0, r1 := validate(4, 1)
		fmt.Printf("%v:%v,%v\n", r7, r0, r1)
	}
}
