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

const uint16_t fifteenBits = 0x7FFF;
const uint16_t fifteenBitMod = 0x8000;

class VM {
   public:
    std::vector<uint16_t> mem;
    std::string filename;
    class Prog {
       public:
        VM *vm;
        virtual void call() = 0;
        virtual inline ~Prog() {}
        Prog(VM *vm) : vm(vm){};
    };
    std::vector<Prog *> functions{
        new F0{this, &F0::_halt}, new F2{this, &F2::_set},
        new F1{this, &F1::_push}, new F1{this, &F1::_pop},
        new F3{this, &F3::_eq},   new F3{this, &F3::_gt},
        new F1{this, &F1::_jmp},  new F2{this, &F2::_jt},
        new F2{this, &F2::_jf},   new F3{this, &F3::_add},
        new F3{this, &F3::_mult}, new F3{this, &F3::_mod},
        new F3{this, &F3::_and},  new F3{this, &F3::_or},
        new F2{this, &F2::_not},  new F2{this, &F2::_rmem},
        new F2{this, &F2::_wmem}, new F1{this, &F1::_call},
        new F0{this, &F0::_ret},  new F1{this, &F1::_out},
        new F1{this, &F1::_in},   new F0{this, &F0::_noop}};

    class F0 : public Prog {
       public:
        F0(VM *vm, void (VM::F0::*fp)()) : Prog(vm), fp(fp){};
        void call() { (this->*fp)(); };
        void (VM::F0::*fp)();
        void _halt(){};
        // stop execution and terminate the prog
        void _noop(){};
        // no operation
        void _ret() {
            vm->ins() = vm->mem.back();
            vm->mem.pop_back();
            // remove the top element from the stack and jump to it; empty stack
            // = halt
        }
    };
    class F1 : public Prog {
       public:
        F1(VM *vm, void (VM::F1::*fp)(uint16_t)) : Prog(vm), fp(fp){};
        void call() {
            uint16_t a = vm->nextI();
            std::cerr << "( " << a << " )\n";
            (this->*fp)(a);
        }
        void (VM::F1::*fp)(uint16_t);
        void _push(uint16_t a) {
            // push <a> onto the stack
            vm->mem.push_back(vm->rv(a));
        }
        void _pop(uint16_t a) {
            // remove the top element from the stack and write it into <a>;
            // empty stack = error
            if (vm->mem.size() <= vm->mem[Stack]) {
                std::cerr << "underflow\n";
                exit(1);
            }
            vm->pro(a) = vm->mem.back();
            vm->mem.pop_back();
        }
        void _jmp(uint16_t a) {
            // jump to <a>
            vm->ins() = vm->rv(a);
        }
        void _call(uint16_t a) {
            // write the address of the next instruction to the stack and jump
            // to <a>
            vm->mem.push_back(vm->ins());
            vm->ins() = vm->rv(a);
        }
        void _out(uint16_t a) {
            // write the character represented by ascii code <a> to the terminal
            std::cout << char(vm->rv(a));
        }
        void _in(uint16_t a) {
            // read a character from the terminal and write its ascii code to
            // <a>; it can be assumed that once input starts, it will continue
            // until a newline is encountered; this means that you can safely
            // read whole lines from the keyboard and trust that they will be
            // fully read
            char c;
            std::cin >> c;
            vm->pro(a) = c;
        }
    };
    class F2 : public Prog {
       public:
        F2(VM *vm, void (VM::F2::*fp)(uint16_t, uint16_t)) : Prog(vm), fp(fp){};
        void call() {
            uint16_t a = vm->nextI();
            uint16_t b = vm->nextI();
            std::cerr << "( " << a << ", " << b << " )\n";
            (this->*fp)(a, b);
        }
        void (VM::F2::*fp)(uint16_t, uint16_t);
        void _set(uint16_t a, uint16_t b) {
            // set register <a> to the value of <b>
            vm->pro(a) = vm->rv(b);
        }
        void _jt(uint16_t a, uint16_t b) {
            // if <a> is nonzero, jump to <b>
            if (vm->rv(a)) vm->ins() = vm->rv(b);
        }
        void _jf(uint16_t a, uint16_t b) {
            // if <a> is zero, jump to <b>
            if (!vm->rv(a)) vm->ins() = vm->rv(b);
        }
        void _not(uint16_t a, uint16_t b) {
            // stores 15-bit bitwise inverse of <b> in <a>
            vm->pro(a) = (~(vm->rv(b))) & fifteenBits;
        }
        void _rmem(uint16_t a, uint16_t b) {
            // read memory at address <b> and write it to <a>
            vm->pro(a) = vm->pro(vm->rv(b));
        }
        void _wmem(uint16_t a, uint16_t b) {
            // write the value from <b> into memory at address <a>
            vm->pro(vm->rv(a)) = vm->rv(b);
        }
    };
    class F3 : public Prog {
       public:
        F3(VM *vm, void (VM::F3::*fp)(uint16_t, uint16_t, uint16_t))
            : Prog(vm), fp(fp){};
        void call() {
            uint16_t a = vm->nextI();
            uint16_t b = vm->nextI();
            uint16_t c = vm->nextI();
            std::cerr << "( " << a << ", " << b << ", " << c << " )\n";
            (this->*fp)(a, b, c);
        }
        void (VM::F3::*fp)(uint16_t, uint16_t, uint16_t);
        void _add(uint16_t a, uint16_t b, uint16_t c) {
            // assign into <a> the sum of <b> and <c> (modulo 32768)
            vm->pro(a) = uint16_t(uint32_t(vm->rv(b)) + uint32_t(vm->rv(c))) %
                         fifteenBitMod;
        }
        void _mult(uint16_t a, uint16_t b, uint16_t c) {
            // store into <a> the product of <b> and <c> (modulo 32768)
            vm->pro(a) = uint16_t(uint32_t(vm->rv(b)) * uint32_t(vm->rv(c))) %
                         fifteenBitMod;
        }
        void _mod(uint16_t a, uint16_t b, uint16_t c) {
            vm->pro(a) = uint16_t(uint32_t(vm->rv(b)) % uint32_t(vm->rv(c))) %
                         fifteenBitMod;
            // store into <a> the remainder of <b> divided by <c>
        }
        void _and(uint16_t a, uint16_t b, uint16_t c) {
            //  stores into <a> the bitwise and of <b> and <c>
            vm->pro(a) = uint16_t(uint32_t(vm->rv(b)) & uint32_t(vm->rv(c))) %
                         fifteenBitMod;
        }
        void _or(uint16_t a, uint16_t b, uint16_t c) {
            //  stores into <a> the bitwise and of <b> and <c>
            vm->pro(a) = uint16_t(uint32_t(vm->rv(b)) | uint32_t(vm->rv(c))) %
                         fifteenBitMod;
            // stores into <a> the bitwise or of <b> and <c>
        }
        void _eq(uint16_t a, uint16_t b, uint16_t c) {
            // set <a> to 1 if <b> is equal to <c>; set it to 0 otherwise
            vm->pro(a) = vm->rv(b) == vm->rv(c) ? 1 : 0;
        }
        void _gt(uint16_t a, uint16_t b, uint16_t c) {
            // set <a> to 1 if <b> is greater than <c>; set it to 0 otherwise
            vm->pro(a) = vm->rv(b) > vm->rv(c) ? 1 : 0;
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
            std::cerr << "Size: " << mem.size() << "\n";
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
    }
    uint16_t rv(uint16_t r) {
        if (r >= 32768) {
            return mem[mem[Registers] + r];
        } else {
            return r;
        }
    }
    uint16_t &pro(uint16_t r) {
        if (r >= 32768) {
            return mem[mem[Registers] + r];
        } else {
            return mem[mem[Program] + r];
        }
    }
    uint16_t &ins() { return mem[Instruction]; }
    uint16_t nextI() { return pro(ins()++); }
    bool step() {
        std::cerr << ins() << " ";
        int instruction = nextI();
        std::cerr << instruction << "\n";
        if (instruction > 21) {
            std::cerr << "bad instruction " << instruction << "\n";
            return false;
        };
        functions[instruction]->call();
        return instruction != 0;
    }
    void run() {
        while (step()) {
        }
    }
};

int main(int argc, char **argv) {
    if (argc != 2) {
        std::cerr << "Usage vm <program.bin>\n";
        exit(1);
    }
    VM *vm = new VM(argv[1]);
    vm->run();
}
