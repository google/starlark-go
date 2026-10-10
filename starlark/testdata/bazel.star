# Benchmarks representative of Bazel workloads, written in pure Starlark
# (no Bazel built-ins): path manipulation (from paths.star) and
# configuration/DAG/template evaluation.

def _path_basename(p):
    return p.rpartition("/")[-1]

def _path_dirname(p):
    prefix, sep, _ = p.rpartition("/")
    if not prefix:
        return sep
    return prefix.rstrip("/")

def _path_is_absolute(path):
    return path.startswith("/") or (len(path) > 2 and path[1] == ":")

def _path_join(path, *others):
    result = path
    for p in others:
        if _path_is_absolute(p):
            result = p
        elif not result or result.endswith("/"):
            result += p
        else:
            result += "/" + p
    return result

def _path_normalize(path):
    if not path:
        return "."
    if path.startswith("//") and not path.startswith("///"):
        initial_slashes = 2
    elif path.startswith("/"):
        initial_slashes = 1
    else:
        initial_slashes = 0
    is_relative = (initial_slashes == 0)

    components = path.split("/")
    new_components = []
    for component in components:
        if component in ("", "."):
            continue
        if component == "..":
            if new_components and new_components[-1] != "..":
                new_components.pop()
            elif is_relative:
                new_components.append(component)
        else:
            new_components.append(component)

    path = "/".join(new_components)
    if not is_relative:
        path = ("/" * initial_slashes) + path
    return path or "."

def _path_relativize(path, start):
    segments = _path_normalize(path).split("/")
    start_segments = _path_normalize(start).split("/")
    if start_segments == ["."]:
        start_segments = []
    start_length = len(start_segments)
    if (path.startswith("/") != start.startswith("/") or
        len(segments) < start_length):
        fail("not beneath")
    for ancestor_segment, segment in zip(start_segments, segments):
        if ancestor_segment != segment:
            fail("not beneath")
    length = len(segments) - start_length
    return "/".join(segments[-length:])

def _path_split_extension(p):
    b = _path_basename(p)
    last_dot = b.rfind(".")
    if last_dot <= 0:
        return (p, "")
    dot_dist = len(b) - last_dot
    return (p[:-dot_dist], p[-dot_dist:])

def _path_replace_extension(p, new_ext):
    return _path_split_extension(p)[0] + new_ext

_sample_paths = [
    "/usr/local//bin/../lib/./python3.11/site-packages/pkg/module.py",
    "src/./internal/../cmd/server/config.yaml",
    "//network/share/dir/./subdir/../file.tar.gz",
    "a/b/c/../../d/e/./f/../g/h.txt",
    "/var/log/nginx/../../spool/mail/root",
    "../../relative/path/./to/../from/archive.zip",
]

def bench_paths(b):
    "Benchmark pure-Starlark path manipulation (paths.star)."
    for _ in range(b.n):
        for p in _sample_paths:
            norm = _path_normalize(p)
            d = _path_dirname(norm)
            base = _path_basename(norm)
            joined = _path_join(d, "sub", "..", base)
            replaced = _path_replace_extension(joined, ".out")
            if d and d != "/":
                _path_relativize(replaced, d)

def _parse_semver(s):
    "Parses 'major.minor.patch[-prerelease]' into a comparable tuple."
    main, _, pre = s.partition("-")
    parts = main.split(".")
    if len(parts) != 3:
        fail("invalid version: " + s)
    return (int(parts[0]), int(parts[1]), int(parts[2]), pre == "", pre)

def _expand_template(tmpl, env):
    "Expands ${KEY} placeholders in tmpl using env dict."
    if "${" not in tmpl:
        return tmpl
    out = []
    rest = tmpl
    for _ in range(len(tmpl)):
        prefix, sep, after = rest.partition("${")
        out.append(prefix)
        if not sep:
            break
        key, sep2, rest = after.partition("}")
        if not sep2:
            fail("unclosed placeholder")
        out.append(str(env.get(key, "")))
    return "".join(out)

def _resolve_services(services, global_env):
    "Topologically sorts services by deps, merges env, and expands templates."
    by_name = {s["name"]: s for s in services}
    indegree = {s["name"]: 0 for s in services}
    dependents = {s["name"]: [] for s in services}
    for s in services:
        name = s["name"]
        for dep in s["deps"]:
            indegree[name] += 1
            dependents[dep].append(name)

    queue = [s["name"] for s in services if indegree[s["name"]] == 0]
    order = []
    # Use index cursor over queue list
    for i in range(len(services)):
        if i >= len(queue):
            fail("cycle in service graph")
        cur = queue[i]
        order.append(cur)
        for nxt in dependents[cur]:
            indegree[nxt] -= 1
            if indegree[nxt] == 0:
                queue.append(nxt)

    resolved = {}
    for name in order:
        spec = by_name[name]
        env = dict(global_env)
        for dep in spec["deps"]:
            dep_info = resolved[dep]
            env[dep.upper() + "_HOST"] = dep_info["host"]
            env[dep.upper() + "_PORT"] = str(dep_info["port"])
        for k, v in spec["env"].items():
            env[k] = _expand_template(v, env)
        ver = _parse_semver(spec["version"])
        if ver >= (1, 0, 0, True, ""):
            tier = "prod"
        else:
            tier = "staging"
        env["TIER"] = tier
        cmd = [_expand_template(arg, env) for arg in spec["cmd"]]
        resolved[name] = {
            "name": name,
            "host": name + ".internal",
            "port": spec["port"],
            "tier": tier,
            "version": ver,
            "env": env,
            "cmd": cmd,
        }
    return [resolved[name] for name in order]

_sample_services = [
    {
        "name": "db",
        "version": "1.4.2",
        "port": 5432,
        "deps": [],
        "env": {"DATA_DIR": "${ROOT}/pgdata", "MAX_CONN": "200"},
        "cmd": ["/bin/postgres", "-D", "${DATA_DIR}", "-p", "5432"],
    },
    {
        "name": "cache",
        "version": "2.1.0",
        "port": 6379,
        "deps": [],
        "env": {"MEM_LIMIT": "512mb"},
        "cmd": ["/bin/redis", "--port", "6379", "--maxmemory", "${MEM_LIMIT}"],
    },
    {
        "name": "auth",
        "version": "1.0.5-rc1",
        "port": 8081,
        "deps": ["db", "cache"],
        "env": {
            "DB_URL": "postgres://${DB_HOST}:${DB_PORT}/auth",
            "CACHE_ADDR": "${CACHE_HOST}:${CACHE_PORT}",
        },
        "cmd": ["/bin/authd", "--db=${DB_URL}", "--cache=${CACHE_ADDR}", "--tier=${TIER}"],
    },
    {
        "name": "billing",
        "version": "1.2.0",
        "port": 8082,
        "deps": ["db", "auth"],
        "env": {
            "DB_URL": "postgres://${DB_HOST}:${DB_PORT}/billing",
            "AUTH_URL": "http://${AUTH_HOST}:${AUTH_PORT}",
        },
        "cmd": ["/bin/billingd", "--db=${DB_URL}", "--auth=${AUTH_URL}"],
    },
    {
        "name": "search",
        "version": "0.9.4",
        "port": 8083,
        "deps": ["db", "cache"],
        "env": {
            "INDEX_PATH": "${ROOT}/index",
            "CACHE_ADDR": "${CACHE_HOST}:${CACHE_PORT}",
        },
        "cmd": ["/bin/searchd", "--index=${INDEX_PATH}", "--cache=${CACHE_ADDR}"],
    },
    {
        "name": "api",
        "version": "2.0.1",
        "port": 8080,
        "deps": ["auth", "billing", "search", "cache"],
        "env": {
            "AUTH_URL": "http://${AUTH_HOST}:${AUTH_PORT}",
            "BILLING_URL": "http://${BILLING_HOST}:${BILLING_PORT}",
            "SEARCH_URL": "http://${SEARCH_HOST}:${SEARCH_PORT}",
        },
        "cmd": ["/bin/apid", "--auth=${AUTH_URL}", "--billing=${BILLING_URL}", "--search=${SEARCH_URL}"],
    },
]

_sample_global_env = {"ROOT": "/var/lib/services", "REGION": "us-east-1"}

def bench_config(b):
    "Benchmark plain-Starlark config evaluation (DAG toposort, template expansion, semver parsing)."
    for _ in range(b.n):
        _resolve_services(_sample_services, _sample_global_env)
