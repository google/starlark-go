# Tests of time module.

load('assert.star', 'assert')
load('time.star', 'time')

assert.eq(time.parse_time("2020-06-26T17:38:36Z"), time.from_timestamp(1593193116))

assert.eq(time.parse_time("1970-01-01T00:00:00Z").unix, 0)
assert.eq(time.parse_time("1970-01-01T00:00:00Z").unix_nano, 0)

t = time.parse_time("2000-01-02T03:04:05Z")
assert.eq(t.year, 2000)
assert.eq(t.in_location("US/Eastern"), time.parse_time("2000-01-01T22:04:05-05:00"))
assert.eq(t.in_location("US/Eastern").format("3 04 PM"), "10 04 PM")

assert.eq(t - t, time.parse_duration("0s"))

d = time.parse_duration("1s")
assert.eq(d - d, time.parse_duration("0"))
assert.eq(d + d, time.parse_duration("2s"))
assert.eq(d * 5, time.parse_duration("5s"))
assert.eq(time.parse_duration("0s") + time.parse_duration("3m35s"), time.parse_duration("3m35s"))

d2 = time.parse_duration("10h")
# duration attributes
assert.eq(10.0, d2.hours)
assert.eq(10*60.0, d2.minutes)
assert.eq(10*60*60.0, d2.seconds)
assert.eq(10*60*60*1000, d2.milliseconds)
assert.eq(10*60*60*1000000, d2.microseconds)
assert.eq(10*60*60*1000000000, d2.nanoseconds)

# duration == duration = boolean
# duration != duration = boolean
assert.eq(time.parse_duration("1h"), time.parse_duration("1h"))
assert.ne(time.parse_duration("1h"), time.parse_duration("1m"))
# duration < duration = boolean
assert.lt(time.parse_duration("1m"), time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") < time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") < time.parse_duration("1m"))
# duration <= duration = boolean
assert.true(time.parse_duration("1m") <= time.parse_duration("1h"))
assert.true(time.parse_duration("1h") <= time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") <= time.parse_duration("1m"))
# duration > duration = boolean
assert.true(not time.parse_duration("1m") > time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") > time.parse_duration("1h"))
assert.true(time.parse_duration("1h") > time.parse_duration("1m"))
# duration >= duration = boolean
assert.true(not time.parse_duration("1m") >= time.parse_duration("1h"))
assert.true(time.parse_duration("1h") >= time.parse_duration("1h"))
assert.true(time.parse_duration("1h") >= time.parse_duration("1m"))

# duration + duration = duration
assert.eq(d2 + d, time.parse_duration("10h01s"))
# duration + time = time
assert.eq(d2 + time.parse_time("2011-04-22T13:33:48Z"), time.parse_time("2011-04-22T23:33:48Z"))
# duration - duration = duration
assert.eq(d2 - d, time.parse_duration("9h59m59s"))
# duration / duration = float
assert.eq(d2 / time.parse_duration("16m"), 37.5)
assert.fails(lambda: d2 / time.parse_duration("0"), "division by zero")
# duration / int = duration
assert.eq(d2 / 20, time.parse_duration("30m"))
assert.fails(lambda: d2 / 0, "division by zero")
# duration // duration = int
assert.eq(d2 // time.parse_duration("16m"), 37)
assert.fails(lambda: d2 // time.parse_duration("0"), "division by zero")
# duration * int = duration
assert.eq(d * 1000, time.parse_duration("16m40s"))
assert.fails(lambda: d2 // 0, "division by zero")


before = time.now()
time.sleep(10 * time.millisecond)
assert.true((time.now() - before).nanoseconds >= time.parse_duration("10ms").nanoseconds)

# time(year=..., month=..., day=..., hour=..., minute=..., second=..., nanosecond=..., location=...)
t1 = time.time(2009, 6, 12, 12, 6, 10, 99, "US/Eastern")
assert.eq(t1, time.parse_time("2009-06-12T12:06:10.000000099", format="2006-01-02T15:04:05.999999999", location="US/Eastern"))
assert.eq(time.time(year=2012, month=12, day=31), time.parse_time("2012-12-31T00:00:00Z"))

# time attributes
assert.eq(2009, t1.year)
assert.eq(6, t1.month)
assert.eq(12, t1.day)
assert.eq(12, t1.hour)
assert.eq(6, t1.minute)
assert.eq(10, t1.second)
assert.eq(99, t1.nanosecond)
assert.eq(1244822770, t1.unix)
assert.eq(1244822770000000099, t1.unix_nano)

# time == time = boolean
# time != time = boolean
assert.eq(time.parse_time("2011-04-22T13:33:48Z"), time.parse_time("2011-04-22T13:33:48Z"))
assert.ne(time.parse_time("2011-04-22T13:33:48Z"), time.parse_time("2011-04-22T13:33:49Z"))
# time < time = boolean
assert.lt(time.parse_time("2010-04-22T13:33:48Z"), time.parse_time("2011-04-22T13:33:48Z"))
assert.true(not time.parse_time("2010-04-22T13:33:48Z") < time.parse_time("2010-04-22T13:33:48Z"))
assert.true(not time.parse_time("2010-04-22T13:33:48Z") < time.parse_time("2009-04-22T13:33:48Z"))
# time <= time = boolean
assert.true(time.parse_time("2010-04-22T13:33:48Z") <= time.parse_time("2011-04-22T13:33:48Z"))
assert.true(time.parse_time("2010-04-22T13:33:48Z") <= time.parse_time("2010-04-22T13:33:48Z"))
assert.true(not time.parse_time("2010-04-22T13:33:48Z") <= time.parse_time("2009-04-22T13:33:48Z"))
# time > time = boolean
assert.true(time.parse_time("2012-04-22T13:33:48Z") > time.parse_time("2011-04-22T13:33:48Z"))
assert.true(not time.parse_time("2011-04-22T13:33:48Z") > time.parse_time("2011-04-22T13:33:48Z"))
assert.true(not time.parse_time("2010-04-22T13:33:48Z") > time.parse_time("2011-04-22T13:33:48Z"))
# time >= time = boolean
assert.true(time.parse_time("2012-04-22T13:33:48Z") >= time.parse_time("2011-04-22T13:33:48Z"))
assert.true(time.parse_time("2011-04-22T13:33:48Z") >= time.parse_time("2011-04-22T13:33:48Z"))
assert.true(not time.parse_time("2010-04-22T13:33:48Z") >= time.parse_time("2011-04-22T13:33:48Z"))
# time + duration = time
assert.eq(time.parse_time("2011-04-22T13:33:48Z") + time.parse_duration("10h"), time.parse_time("2011-04-22T23:33:48Z"))
# time - duration = time
assert.eq(time.parse_time("2011-04-22T13:33:48Z") - time.parse_duration("10h"), time.parse_time("2011-04-22T03:33:48Z"))
# time - time = duration
assert.eq(time.parse_time("2011-04-22T13:33:48Z") - time.parse_time("2011-04-22T03:33:48Z"), time.parse_duration("10h"))
