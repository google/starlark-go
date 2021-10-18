load("regexp.star", "regexp")
load("assert.star", "assert")

# bad regular expression
assert.fails(lambda: regexp.compile("a(b"), "error parsing regexp")
# regular expression with the forbidden byte-oriented feature
assert.fails(lambda: regexp.compile(".\\C+"), "error parsing regexp")
assert.fails(lambda: regexp.compile('.\\C+'), "error parsing regexp")
assert.fails(lambda: regexp.compile(r'.\C+'), "error parsing regexp")
assert.fails(lambda: regexp.compile(r'.\\\C+'), "error parsing regexp")
assert.fails(lambda: regexp.compile("""
.\\C+
"""), "error parsing regexp")

re_abs = regexp.compile("ab+")
re_multi_patterns = regexp.compile("ab+|a.*c")
re_ax = regexp.compile("a.")
re_with_sub_matches = regexp.compile("a(x*)b(y|z)c")
re_axsb = regexp.compile("a(x*)b")
re_a = regexp.compile("a")
re_zs = regexp.compile("z+")
re_mls = regexp.compile("(?m)(\\w+):\\s+(\\w+)$")
re_3no = regexp.compile(r'^\d{3}$')
re_bc = regexp.compile(r'\\C')
re_mgroups = regexp.compile(r'(\w+)-(\d+)-(\w+)')
re_ds = regexp.compile(r"\d*")

# matches
assert.true(re_3no.matches("123"))
assert.true(not re_3no.matches("1234"))
assert.true(not re_3no.matches("12a"))
assert.true(re_abs.matches("ab"))
assert.true(re_abs.matches("abbbb"))
assert.true(re_abs.matches("cab"))
assert.true(not re_abs.matches("a"))
assert.true(not re_abs.matches("ca"))
assert.true(not re_abs.matches("ba"))

# find
assert.eq("ab", re_abs.find("ab"))
assert.eq("abbbb", re_abs.find("abbbb"))
assert.eq("ab", re_abs.find("cab"))
assert.eq(None, re_abs.find("a"))
assert.eq(None, re_abs.find("ca"))
assert.eq(None, re_abs.find("ba"))
assert.eq("abbbb", re_multi_patterns.find("abbbbc"))

# find_index
assert.eq([0, 2], re_abs.find_index("ab"))
assert.eq([0, 5], re_abs.find_index("abbbb"))
assert.eq([1, 3], re_abs.find_index("cab"))
assert.eq(None, re_abs.find_index("a"))
assert.eq(None, re_abs.find_index("ca"))
assert.eq(None, re_abs.find_index("ba"))
assert.eq([0, 5], re_multi_patterns.find_index("abbbbc"))

# find_all
assert.eq(["ar", "an", "al"], re_ax.find_all("paranormal"))
assert.eq(["aa"], re_ax.find_all("graal"))
assert.eq([], re_ax.find_all("none"))

# find_all_index
assert.eq([[1, 3], [3, 5], [8, 10]], re_ax.find_all_index("paranormal"))
assert.eq([[2, 4]], re_ax.find_all_index("graal"))
assert.eq([], re_ax.find_all_index("none"))

# find_submatch
assert.eq(["axxxbyc", "xxx", "y"], re_with_sub_matches.find_submatch("-axxxbyc-"))
assert.eq(["abzc", "", "z"], re_with_sub_matches.find_submatch("-abzc-"))
assert.eq(None, re_with_sub_matches.find_submatch("none"))
assert.eq(["key: value", "key", "value"], re_mls.find_submatch("""
	# comment line
	key: value
"""))

# find_submatch_index
assert.eq([1, 8, 2, 5, 6, 7], re_with_sub_matches.find_submatch_index("-axxxbyc-"))
assert.eq([1, 5, 2, 2, 3, 4], re_with_sub_matches.find_submatch_index("-abzc-"))
assert.eq(None, re_with_sub_matches.find_submatch_index("none"))
assert.eq([18, 28, 18, 21, 23, 28], re_mls.find_submatch_index("""
	# comment line
	key: value
"""))

# find_all_submatch
assert.eq([["ab", ""]], re_axsb.find_all_submatch("-ab-"))
assert.eq([["axxb", "xx"]], re_axsb.find_all_submatch("-axxb-"))
assert.eq([["ab", ""], ["axb", "x"]], re_axsb.find_all_submatch("-ab-axb-"))
assert.eq([["axxb", "xx"], ["ab", ""]], re_axsb.find_all_submatch("-axxb-ab-"))
assert.eq([], re_axsb.find_all_submatch("none"))

# find_all_submatch_index
assert.eq([[1, 3, 2, 2]], re_axsb.find_all_submatch_index("-ab-"))
assert.eq([[1, 5, 2, 4]], re_axsb.find_all_submatch_index("-axxb-"))
assert.eq([[1, 3, 2, 2], [4, 7, 5, 6]], re_axsb.find_all_submatch_index("-ab-axb-"))
assert.eq([[1, 5, 2, 4], [6, 8, 7, 7]], re_axsb.find_all_submatch_index("-axxb-ab-"))
assert.eq([], re_axsb.find_all_submatch_index("none"))

# replace_all with unexpected type
assert.fails(lambda: re_axsb.replace_all("-ab-axxb-", 12), "got int")

# replace_all with string
assert.eq("-T-T-", re_axsb.replace_all("-ab-axxb-", "T"))
assert.eq("--xx-", re_axsb.replace_all("-ab-axxb-", "$1"))
assert.eq("---", re_axsb.replace_all("-ab-axxb-", "$1W"))
assert.eq("-W-xxW-", re_axsb.replace_all("-ab-axxb-", "${1}W"))
assert.eq("none", re_axsb.replace_all("none", "X"))
assert.eq("-T-T-", re_bc.replace_all(r'-\C-\C-', "T"))

def toUpperCase(src):
  return src.upper()

# replace_all with function
assert.eq("cABcAXXBc", re_axsb.replace_all("cabcaxxbc", toUpperCase))
assert.eq("cABcAXXBc", re_axsb.replace_all("cabcaxxbc", lambda src: src.upper()))
assert.fails(lambda: re_axsb.replace_all("cabcaxxbc", lambda src: src.none), "string has no .none")
assert.fails(lambda: re_axsb.replace_all("cabcaxxbc", lambda src: 1), "returned int, want string")

# replace_all_literal
assert.eq("-T-T-", re_axsb.replace_all_literal("-ab-axxb-", "T"))
assert.eq("-$1-$1-", re_axsb.replace_all_literal("-ab-axxb-", "$1"))
assert.eq("-${1}-${1}-", re_axsb.replace_all_literal("-ab-axxb-", "${1}"))
assert.eq("none", re_axsb.replace_all_literal("none", "X"))
assert.eq("-T-T-", re_bc.replace_all_literal(r'-\C-\C-', "T"))

# split
assert.eq(["b", "n", "n", ""], re_a.split("banana"))
assert.eq(["b", "n", "n", ""], re_a.split("banana", max=-1))
assert.eq(["b", "n", "n", ""], re_a.split("banana", max=-10))
assert.eq(["b", "n", "n", ""], re_a.split("banana", max=0))
assert.eq(["banana"], re_a.split("banana", max=1))
assert.eq(["b", "nana"], re_a.split("banana", max=2))
assert.eq(["pi", "a"], re_zs.split("pizza"))
assert.eq(["pi", "a"], re_zs.split("pizza", max=-1))
assert.eq(["pi", "a"], re_zs.split("pizza", max=-10))
assert.eq(["pi", "a"], re_zs.split("pizza", max=0))
assert.eq(["pizza"], re_zs.split("pizza", max=1))
assert.eq(["pi", "a"], re_zs.split("pizza", max=2))
assert.eq(["pi", "a"], re_zs.split("pizza", max=20))
assert.eq(["b", "a", "n", "a", "n", "a"], re_ds.split("banana", max=-1))
