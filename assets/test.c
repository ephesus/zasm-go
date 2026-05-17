//checking ASLR
//if compiled with -g for gdb, aslr is disabled
//use gcc -o test test.asm (without the -g)

#include <stdio.h>

int main() {
    int stack_var = 0;
    int stack_var2 = 0;

    printf("Stack address: %p\n", (void*)&stack_var);
    printf("Stack address: %p\n", (void*)&stack_var2);

    return 0;
}
