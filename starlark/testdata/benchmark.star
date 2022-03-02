# Benchmarks of Starlark execution

def bench_range_construction(b):
    for _ in range(b.n):
        range(200)

def bench_range_iteration(b):
    for _ in range(b.n):
        for x in range(200):
            pass

# Make a 2-level call tree of 100 * 100 calls.
def bench_calling(b):
    list = range(100)

    def g():
        for x in list:
            pass

    def f():
        for x in list:
            g()

    for _ in range(b.n):
        f()

# Measure overhead of calling a trivial built-in method.
emptydict = {}
range1000 = range(1000)

def bench_builtin_method(b):
    for _ in range(b.n):
        for _ in range1000:
            emptydict.get(None)

def bench_int(b):
    for _ in range(b.n):
        a = 0
        for _ in range1000:
            a += 1

def bench_bigint(b):
    for _ in range(b.n):
        a = 1 << 31  # maxint32 + 1
        for _ in range1000:
            a += 1

def bench_gauss(b):
    # Sum of arithmetic series. All results fit in int32.
    for _ in range(b.n):
        acc = 0
        for x in range(92000):
            acc += x

def bench_mix(b):
    "Benchmark of a simple mix of computation (for, if, arithmetic, comprehension)."
    for _ in range(b.n):
        x = 0
        for i in range(50):
            if i:
                x += 1
            a = [x for x in range(i)]

largedict = {str(v): v for v in range(1000)}

def bench_dict_equal(b):
    "Benchmark of dict equality operation."
    for _ in range(b.n):
        if largedict != largedict:
            fail("invalid comparison")
