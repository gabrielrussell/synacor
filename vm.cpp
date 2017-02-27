#include <stdlib.h>
#include <cstdint>
#include <fstream>
#include <iostream>
#include <string>
#include <vector>

enum MemRegion {
    Instruction,
    Registers,
    Program,
    Stack,
};

typedef void fp0_t();
typedef void fp1_t(uint16_t);
typedef void fp2_t(uint16_t, uint16_t);
typedef void fp3_t(uint16_t, uint16_t, uint16_t);

class VM {
   public:
    std::vector<uint16_t> mem;
    std::string filename;
    class Prog {
       public:
        virtual void call(uint16_t (*)()) = 0;
        virtual inline ~Prog() {}
        VM *vm;
    };
    std::vector<Prog*> functions{
        new F0{&F0::_halt},
        new F2{&F2::_set},
        new F1{&F1::_push},
        new F1{&F1::_pop},
        new F3{&F3::_eq },
        new F3{&F3::_gt },
        new F1{&F1::_jmp },
        new F2{&F2::_jt },
        new F2{&F2::_jf },
        new F3{&F3::_add },
        new F3{&F3::_mult },
        new F3{&F3::_mod },
        new F3{&F3::_and },
        new F3{&F3::_or },
        new F2{&F2::_not },
        new F2{&F2::_rmem },
        new F2{&F2::_wmem },
        new F1{&F1::_call },
        new F0{&F0::_ret },
        new F1{&F1::_out },
        new F1{&F1::_in },
        new F0{&F0::_noop }
    };

    class F0 : public Prog {
       public:
        F0(void (VM::F0::*fp)()) : fp(fp){};
        void call(uint16_t (*nextI)()) { (this->*fp)(); };
        void (VM::F0::*fp)();
        void _halt(){};
        // stop execution and terminate the prog
        void _noop(){};
        // no operation
        void _ret() {}
        // remove the top element from the stack and jump to it; empty stack =
    };
    class F1 : public Prog {
       public:
        F1(void (VM::F1::*fp)(uint16_t)) : fp(fp){};
        void call(uint16_t (*nextI)()) { (this->*fp)(nextI()); }
        void (VM::F1::*fp)(uint16_t);
        void _push(uint16_t a) {
            // push <a> onto the stack
            vm->mem.push_back(a);
        }
        void _pop(uint16_t a) {
            // remove the top element from the stack and write it into <a>;
            // empty
            // stack = error
            if (vm->mem.size() <= vm->mem[Stack]) {
                std::cerr << "underflow\n";
                exit(1);
            }
            *(vm->prog(a)) = vm->mem.back();
            vm->mem.pop_back();
        }
        void _jmp(uint16_t a) {}
        // jump to <a>
        void _call(uint16_t a) {}
        // write the address of the next instruction to the stack and jump to
        // <a>
        // halt
        void _out(uint16_t a) { std::cout << char(a); }
        // write the character represented by ascii code <a> to the terminal
        void _in(uint16_t a) {
            char c;
            std::cin >> c;
        }
        // read a character from the terminal and write its ascii code to <a>;
        // it
        // can be assumed that once input starts, it will continue until a
        // newline
        // is encountered; this means that you can safely read whole lines from
        // the
        // keyboard and trust that they will be fully read
    };
    class F2 : public Prog {
       public:
        F2(void (VM::F2::*fp)(uint16_t,uint16_t)) : fp(fp){};
        void call(uint16_t (*nextI)()) { (this->*fp)(nextI(), nextI()); }
        void (VM::F2::*fp)(uint16_t, uint16_t);
        void _set(uint16_t a, uint16_t b) {
            // set register <a> to the value of <b>
            *(vm->prog(a)) = b;
        }
        void _jt(uint16_t a, uint16_t b) {}
        // if <a> is nonzero, jump to <b>
        void _jf(uint16_t a, uint16_t b) {}
        // if <a> is zero, jump to <b>
        void _not(uint16_t a, uint16_t b) {}
        // stores 15-bit bitwise inverse of <b> in <a>
        void _rmem(uint16_t a, uint16_t b) {}
        // read memory at address <b> and write it to <a>
        void _wmem(uint16_t a, uint16_t b) {}
        // write the value from <b> into memory at address <a>
    };
    class F3 : public Prog {
       public:
        F3(void (VM::F3::*fp)(uint16_t,uint16_t,uint16_t)) : fp(fp){};
        void call(uint16_t (*nextI)()) { (this->*fp)(nextI(), nextI(), nextI()); }
        void (VM::F3::*fp)(uint16_t, uint16_t, uint16_t);
        void _add(uint16_t a, uint16_t b, uint16_t c) {}
        // assign into <a> the sum of <b> and <c> (modulo 32768)
        void _mult(uint16_t a, uint16_t b, uint16_t c) {}
        // store into <a> the product of <b> and <c> (modulo 32768)
        void _mod(uint16_t a, uint16_t b, uint16_t c) {}
        // stores into <a> the bitwise and of <b> and <c>
        void _and(uint16_t a, uint16_t b, uint16_t c) {}
        //  stores into <a> the bitwise and of <b> and <c>
        void _or(uint16_t a, uint16_t b, uint16_t c) {}
        // stores into <a> the bitwise or of <b> and <c>
        void _eq(uint16_t a, uint16_t b, uint16_t c) {
            // set <a> to 1 if <b> is equal to <c>; set it to 0 otherwise
            *(vm->prog(a)) = *(vm->prog(b)) == *(vm->prog(c)) ? 1 : 0;
        }
        void _gt(uint16_t a, uint16_t b, uint16_t c) {
            // set <a> to 1 if <b> is greater than <c>; set it to 0 otherwise
        }
    };
    void load() {
        try {
            std::ifstream input(filename, std::ios::in | std::ios::binary);
            if (!input.is_open() || !input.good()) {
                std::cerr << "File not open\n";
                return;
            }
            uint16_t d;
            while (!input.eof()) {
                input.read(reinterpret_cast<char *>(&d), 2);
                mem.push_back(d);
            }
            std::cout << "Size: " << mem.size() << "\n";
        } catch (std::ifstream::failure e) {
            std::cerr << "Excepion reading file "
                      << "\n";
        }
    }
    VM(std::string filename) : filename(filename) {
        mem.push_back(0);  // Instruction
        mem.push_back(0);  // Registers
        mem.push_back(0);  // Program
        mem.push_back(0);  // Stack
        mem[Registers] = mem.size();
        for (int i = 0; i < 8; i++) {
            mem.push_back(0);
        }
        mem[Program] = mem.size();
        load();
        mem[Stack] = mem.size();
        mem[Instruction] = mem[Program];
        //functions.push_back(new F0{&F0::_halt});
    }
    void run() {
        bool halt = false;
        while (!halt) {
            halt = step();
        }
    }
    uint16_t *reg(uint16_t r) { return &mem[mem[Registers] + r]; }
    uint16_t *prog(uint16_t r) { return &mem[mem[Program] + r]; }
    uint16_t nextI() { return mem[mem[Instruction]++]; }
    bool step() {
        if (mem[mem[Instruction]] == 0) {
            return false;
        }
        // int instruction = nextI();
        return true;
    }
};

int main(int argc, char **argv) {
    if (argc != 2) {
        std::cerr << "Usage vm <program.bin>\n";
        exit(1);
    }
    VM _vm(argv[1]);
    VM *vm = &_vm;
    vm->run();
}
