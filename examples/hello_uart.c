// Assuming quemu virt UART0
#define UART0_ADDR 0x10000000

void uart_putc(char c) {
    volatile char *uart = (char *)UART0_ADDR;
    *uart = c;
}

void _main() {
    uart_putc('H');
    uart_putc('e');
    uart_putc('l');
    uart_putc('l');
    uart_putc('o');
    uart_putc(',');
    uart_putc(' ');
    uart_putc('W');
    uart_putc('o');
    uart_putc('r');
    uart_putc('l');
    uart_putc('d');
    uart_putc('!');
}
