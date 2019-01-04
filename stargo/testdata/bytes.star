# An example of using the bytes.Buffer type.
	
load("go", "bytes")
load("assert.star", "assert")

# A pointer to a bytes.Buffer.
b = go.new(bytes.Buffer)
assert.eq(type(b), 'go.ptr<*bytes.Buffer>')
assert.eq(b.WriteString("hi"), (2, None))
assert.eq(b.String(), 'hi')
assert.eq(b.WriteString(", again"), (7, None))
assert.eq(b.String(), "hi, again")
assert.eq(str(b), "hi, again") # calls String method

assert.contains(dir(b), 'WriteString') # an exported method
assert.contains(dir(b), 'buf') # a private field
assert.fails(lambda: b.buf, 'access to unexported field .buf')

b.Reset()
b.WriteString("abc")
b.WriteString(b.Bytes()) # []byte is implicitly converted to string
b.Write(b.String()) # string is implicitly converted to []byte
assert.eq(b.String(), 'abcabcabcabc')

# A value of type bytes.Buffer
b = bytes.Buffer() # no 'new'
assert.eq(type(b), 'go.struct<bytes.Buffer>')
assert.eq(str(b), '{[] 0 0}') # no String method
assert.fails(lambda: b.WriteString(""), 'go.struct<bytes.Buffer> has no .WriteString field or method')

