package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"vm"
)

func calc(x []int) int {
	a, b, c, d, e := x[0], x[1], x[2], x[3], x[4]
	return a + b*c*c + d*d*d - e
}

func swap(x []string, a, b int) {
	if a == b {
		return
	}
	tmp := x[b]
	x[b] = x[a]
	x[a] = tmp
}

func comb(x []string, y int, r chan []string) {
	if y == len(x) {
		z := make([]string, len(x))
		copy(z, x)
		r <- z
		return
	}
	for c := y; c < len(x); c++ {
		swap(x, y, c)
		comb(x, y+1, r)
		swap(x, y, c)
	}
}

func combine(x []string) chan []string {
	r := make(chan []string)
	go func() {
		comb(x, 0, r)
		close(r)
	}()
	return r
}

func main() {

	cChan := combine([]string{
		"red",
		"corroded",
		"shiny",
		"concave",
		"blue",
	})

	var err error

	flag.Parse()

	for {
		c, ok := <-cChan
		if !ok {
			break
		}
		in := bytes.NewBuffer([]byte{})
		out := bytes.NewBuffer([]byte{})
		for _, coin := range c {
			fmt.Fprintf(in, "use %v coin\n", coin)
		}
		v := &vm.VM{
			Stdout: out,
			Stdin:  in,
		}
		err = v.LoadVM(flag.Arg(0))
		if err != nil {
			fmt.Printf("load error %v\n", err)
		}
		fmt.Printf("%v\n", c)
		err = v.Run()
		if err != nil && err != io.EOF {
			fmt.Printf("program error %v\n", err)
			os.Exit(1)
		}
		if strings.Contains(out.String(), "click") {
			fmt.Printf("%v", out.String())
			v.SaveVM("unlocked")
		}
	}

}
