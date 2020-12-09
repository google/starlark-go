load('time', 'time')
load('assert.star', 'assert')

assert.eq(time.parse_time("2011-04-22T13:33:48Z"), time.parse_time("2011-04-22T13:33:48Z"))
assert.eq(time.zero, time.parse_time("0001-01-01T00:00:00Z"))
assert.true(time.parse_time("2010-04-22T13:33:48Z") < time.parse_time("2011-04-22T13:33:48Z"))
assert.true(time.parse_time("2011-04-22T13:33:48Z") == time.parse_time("2011-04-22T13:33:48Z"))
assert.true(time.parse_time("2012-04-22T13:33:48Z") > time.parse_time("2011-04-22T13:33:48Z"))

assert.eq(time.parse_time("2020-06-26T17:38:36Z"), time.from_timestamp(1593193116))

# zero
assert.eq(time.zero.format("Mon Jan 2 15:04:05 -0700 MST 2006"), "Mon Jan 1 00:00:00 +0000 UTC 0001")
assert.eq(time.parse_time("1970-01-01T00:00:00Z").unix, 0)
assert.eq(time.parse_time("1970-01-01T00:00:00Z").unix_nano, 0)

t = time.parse_time("2000-01-02T03:04:05Z")
assert.eq(t.year, 2000)
assert.eq(t.in_location("US/Eastern"), time.parse_time("2000-01-01T22:04:05-05:00"))
assert.eq(t.in_location("US/Eastern").format("3 04 PM"), "10 04 PM")

assert.eq(t - t, time.parse_duration("0s"))

d = time.parse_duration("1s")
assert.eq(d - d, time.duration(0))
assert.eq(d + d, time.parse_duration("2s"))
assert.eq(d * 5, time.parse_duration("5s"))
assert.eq(time.parse_duration("0s") + time.parse_duration("3m35s"), time.parse_duration("3m35s"))

d2 = time.parse_duration("10h")
assert.eq(10.0, d2.hours)
assert.eq(10*60.0, d2.minutes)
assert.eq(10*60*60.0, d2.seconds)
assert.eq(10*60*60*1000000000, d2.nanoseconds)


time.sleep(time.second)
