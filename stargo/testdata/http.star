# An example of using the net/http package.
# TODO: non-hermetic!

load("go", http="net/http", ioutil="io/ioutil")
load("assert.star", "assert")

def main():
    resp, err = http.Get("http://golang.org/")
    if err != None:
      print(err) # typically a go.ptr<*url.Error>
      return # e.g. offline
      
    assert.eq(resp.Status, "200 OK")
    data, err = ioutil.ReadAll(resp.Body)
    assert.eq(err, None)
    data = go.string(data)
    assert.eq(data[:50], "<!DOCTYPE html>\n<html>\n<head>\n<meta http-equiv=\"Co")

    assert.eq(type(resp.Header), 'go.map<http.Header>')
    assert.eq(go.typeof(resp.Header), http.Header)
    assert.contains(resp.Header.Get('Content-Type'), 'text/html')
    
main()
