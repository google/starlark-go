# Benchmarks of Starlark execution
# option:set

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

largeset = set([v for v in range(1000)])

def bench_set_equal(b):
    "Benchmark of set union operation."
    for _ in range(b.n):
        if largeset != largeset:
            fail("invalid comparison")

flat = { "int": 1, "float": 0.2, "string": "string", "list": [], "bool": True, "nil": None, "tuple": (1, 2, 3) }
deep = {
    "type": "int",
    "value": 1, 
    "next": {
        "type": "float",
        "value": 0.2,
        "next": {
            "type": "string", 
            "value": "string",
            "next": {
                "type": "list", 
                "value": [ 1, "", True, None, (1, 2) ],
                "next": {
                    "type": "bool",
                    "value": True,
                    "next": {
                        "type": "tuple",
                        "value": (1, 2.0, "3"),
                        "next": None
                    }
                }
            }
        }
    }
}

deep_list = [ deep for _ in range(100) ]

def bench_to_json_flat_mixed(b):
    "Benchmark json.encode builtin with flat mixed input"
    for _ in range(b.n):
        json.encode(flat)

def bench_to_json_flat_big(b):
    "Benchmark json.encode builtin with big flat integer input"
    for _ in range(b.n):
        json.encode(largedict)

def bench_to_json_deep(b):
    "Benchmark json.encode builtin with deep input"
    for _ in range(b.n):
        json.encode(deep)

def bench_to_json_deep_list(b):
    "Benchmark json.encode builtin with a list of deep input"
    for _ in range(b.n):
        json.encode(deep)

def bench_issubset_unique_large_small(b):
    "Benchmark set.issubset builtin"
    s = set(range(10000))
    for _ in range(b.n):
        s.issubset(range(1000))

def bench_issubset_unique_small_large(b):
    "Benchmark set.issubset builtin"
    s = set(range(1000))
    for _ in range(b.n):
        s.issubset(range(10000))

def bench_issubset_unique_same(b):
    "Benchmark set.issubset builtin"
    s = set(range(1000))
    for _ in range(b.n):
        s.issubset(range(1000))

def bench_issubset_duplicate_large_small(b):
    "Benchmark set.issubset builtin"
    s = set(range(10000))
    l = list(range(200)) * 5
    for _ in range(b.n):
        s.issubset(range(1000))

def bench_issubset_duplicate_small_large(b):
    "Benchmark set.issubset builtin"
    s = set(range(1000))
    l = list(range(2000)) * 5
    for _ in range(b.n):
        s.issubset(l)

def bench_issubset_duplicate_same(b):
    "Benchmark set.issubset builtin"
    s = set(range(1000))
    l = list(range(200)) * 5
    for _ in range(b.n):
        s.issubset(l)
