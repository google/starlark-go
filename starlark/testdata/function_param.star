# Functions used in starlark_test.TestParamDefault().

def all_required(a, b, c): pass
def all_opt(a="a", b=None, c=""): pass
def mix_required_opt(a, b, c="c", d="d"): pass
def with_varargs(a, b="b", *args): pass
def with_varargs_kwonly(a, b="b", *args, c="c", d): pass
def with_kwonly(a, b="b", *, c="c", d): pass
def with_kwargs(a, b="b", c="c", **kwargs): pass
def with_varargs_kwonly_kwargs(a, b="b", *args, c="c", d, e="e", **kwargs): pass
