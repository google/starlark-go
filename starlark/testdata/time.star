# Tests of time module.

load('assert.star', 'assert')
load('time.star', 'time')

assert.true(time.now() > time.parse_time("2021-03-20T00:00:00Z"))

assert.eq(time.parse_time("2020-06-26T17:38:36Z"), time.from_timestamp(1593193116))
assert.eq(time.parse_time("2020-06-26T17:38:36.123456789", format="2006-01-02T15:04:05.999999999"), time.from_timestamp(1593193116, 123456789))

assert.eq(time.parse_time("1970-01-01T00:00:00Z").unix, 0)
assert.eq(time.parse_time("1970-01-01T00:00:00Z").unix_nano, 0)

t = time.parse_time("2000-01-02T03:04:05Z")
assert.eq(t.year, 2000)
assert.eq(t.in_location("US/Eastern"), time.parse_time("2000-01-01T22:04:05-05:00"))
assert.eq(t.in_location("US/Eastern").format("3 04 PM"), "10 04 PM")

assert.eq(t - t, time.parse_duration("0s"))

d1s = time.parse_duration("1s")
assert.eq(d1s - d1s, time.parse_duration("0"))
assert.eq(d1s + d1s, time.parse_duration("2s"))
assert.eq(d1s * 5, time.parse_duration("5s"))
assert.eq(time.parse_duration("0s") + time.parse_duration("3m35s"), time.parse_duration("3m35s"))

d10h = time.parse_duration("10h")
# duration attributes
assert.eq(10.0, d10h.hours)
assert.eq(10*60.0, d10h.minutes)
assert.eq(10*60*60.0, d10h.seconds)
assert.eq(10*60*60*1000, d10h.milliseconds)
assert.eq(10*60*60*1000000, d10h.microseconds)
assert.eq(10*60*60*1000000000, d10h.nanoseconds)

# duration type
assert.eq("time.duration", type(d10h))
# duration str
assert.eq("10h0m0s", str(d10h))
# duration hash
durations = {
    d10h: "10h",
    d1s: "10s",
}
assert.eq("10h", durations[d10h])
assert.eq("10s", durations[d1s])

# duration == duration
# duration != duration
assert.eq(time.parse_duration("1h"), time.parse_duration("1h"))
assert.ne(time.parse_duration("1h"), time.parse_duration("1m"))
# duration < duration
assert.lt(time.parse_duration("1m"), time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") < time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") < time.parse_duration("1m"))
# duration <= duration
assert.true(time.parse_duration("1m") <= time.parse_duration("1h"))
assert.true(time.parse_duration("1h") <= time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") <= time.parse_duration("1m"))
# duration > duration
assert.true(not time.parse_duration("1m") > time.parse_duration("1h"))
assert.true(not time.parse_duration("1h") > time.parse_duration("1h"))
assert.true(time.parse_duration("1h") > time.parse_duration("1m"))
# duration >= duration
assert.true(not time.parse_duration("1m") >= time.parse_duration("1h"))
assert.true(time.parse_duration("1h") >= time.parse_duration("1h"))
assert.true(time.parse_duration("1h") >= time.parse_duration("1m"))

refTime = time.parse_time("2011-04-22T13:33:48Z")
tenHoursAfterRefTime = time.parse_time("2011-04-22T23:33:48Z")

# duration + duration = duration
assert.eq(d10h + d1s, time.parse_duration("10h01s"))
# duration + time = time
assert.eq(d10h + refTime, tenHoursAfterRefTime)
# duration - duration = duration
assert.eq(d10h - d1s, time.parse_duration("9h59m59s"))
# duration / duration = float
assert.eq(d10h / time.parse_duration("16m"), 37.5)
assert.fails(lambda: d10h / time.parse_duration("0"), "division by zero")
# duration / int = duration
assert.eq(d10h / 20, time.parse_duration("30m"))
assert.fails(lambda: d10h / 0, "division by zero")
# int / duration = error
assert.fails(lambda: 20 / d10h, "unsupported operation")
# duration / float = duration
assert.eq(d10h / 37.5, time.parse_duration("16m"))
assert.fails(lambda: d10h / 0.0, "division by zero")
# duration // duration = int
assert.eq(d10h // time.parse_duration("16m"), 37)
assert.fails(lambda: d10h // time.parse_duration("0"), "division by zero")
# duration * int = duration
assert.eq(d1s * 1000, time.parse_duration("16m40s"))
# int * duration  = duration
assert.eq(1000 * d1s, time.parse_duration("16m40s"))

# is_valid_timezone(location)
assert.true(time.is_valid_timezone("UTC"))
assert.true(time.is_valid_timezone("US/Eastern"))
assert.true(not time.is_valid_timezone("UKN"))

# time(year=..., month=..., day=..., hour=..., minute=..., second=..., nanosecond=..., location=...)
assert.fails(lambda: time.time(2009, 6, 12, 12, 6, 10, 99, "US/Eastern"), "unexpected positional argument")
t1 = time.time(year=2009, month=6, day=12, hour=12, minute=6, second=10, nanosecond=99, location="US/Eastern")
assert.eq(t1, time.parse_time("2009-06-12T12:06:10.000000099", format="2006-01-02T15:04:05.999999999", location="US/Eastern"))
assert.eq(time.time(year=2012, month=12, day=31), time.parse_time("2012-12-31T00:00:00Z"))
assert.eq(time.time(year=2009, month=6, day=12, hour=12, minute=6, second=10, nanosecond=99, location="UTC"), time.time(year=2009, month=6, day=12, hour=12, minute=6, second=10, nanosecond=99))

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

# time type
assert.eq("time.time", type(refTime))
# duration str
assert.eq("2011-04-22 13:33:48 +0000 UTC", str(refTime))
# duration hash
times = {
    refTime: "refTime",
    t1: "t1",
}
assert.eq("refTime", times[refTime])
assert.eq("t1", times[t1])

oneSecondAfterRefTime = time.parse_time("2011-04-22T13:33:49Z")
oneYearAfterRefTime = time.parse_time("2012-04-22T13:33:48Z")
oneYearBeforeRefTime = time.parse_time("2010-04-22T13:33:48Z")
twoYearsBeforeRefTime = time.parse_time("2009-04-22T13:33:48Z")
tenHoursBeforeRefTime = time.parse_time("2011-04-22T03:33:48Z")

# time == time
# time != time
assert.eq(refTime, refTime)
assert.ne(refTime, oneSecondAfterRefTime)
# time < time
assert.lt(oneYearBeforeRefTime, refTime)
assert.true(not oneYearBeforeRefTime < oneYearBeforeRefTime)
assert.true(not oneYearBeforeRefTime < twoYearsBeforeRefTime)
# time <= time
assert.true(oneYearBeforeRefTime <= refTime)
assert.true(oneYearBeforeRefTime <= oneYearBeforeRefTime)
assert.true(not oneYearBeforeRefTime <= twoYearsBeforeRefTime)
# time > time
assert.true(oneYearAfterRefTime > refTime)
assert.true(not refTime > refTime)
assert.true(not oneYearBeforeRefTime > refTime)
# time >= time
assert.true(oneYearAfterRefTime >= refTime)
assert.true(refTime >= refTime)
assert.true(not oneYearBeforeRefTime >= refTime)
# time + duration = time
assert.eq(refTime + d10h, tenHoursAfterRefTime)
# time - duration = time
assert.eq(refTime - d10h, tenHoursBeforeRefTime)
# time - time = duration
assert.eq(refTime - tenHoursBeforeRefTime, d10h)
