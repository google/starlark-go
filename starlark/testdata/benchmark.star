# Benchmarks of Starlark execution
# option:nesteddef

def bench_range():
    return range(200)

# Make a 2-level call tree of 100 * 100 calls.
def bench_calling():
    list = range(100)

    def g():
        for x in list:
            pass

    def f():
        for x in list:
            g()

    f()

# Measure overhead of calling a trivial built-in method.
emptydict = {}
range1000 = range(1000)

def bench_builtin_method():
    for _ in range1000:
        emptydict.get(None)

def bench_int():
    a = 0
    for _ in range1000:
        a += 1

def bench_bigint():
    a = 1 << 31  # maxint32 + 1
    for _ in range1000:
        a += 1

def bench_gauss():
    # Sum of arithmetic series. All results fit in int32.
    acc = 0
    for x in range(92000):
        acc += x
