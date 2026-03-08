int fib(int n) {
	if (n <= 1) {
		return n;
	}

	int a = 0;
	int b = 1;
	int i = 2;
	while (i <= n) {
		int next = a + b;
		a = b;
		b = next;
		i = i + 1;
	}

	return b;
}

int main() {
	return fib(10);
}
