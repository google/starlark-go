# Copyright 2014 The Bazel Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# (From https://github.com/bazelbuild/rules_go/blob/master/go/def.bzl@a6f9d0c)

load("//go/private:repositories.bzl", "go_repositories")
load("//go/private:go_repository.bzl", "go_repository", "new_go_repository")
load("//go/private:go_prefix.bzl", "go_prefix")
load("//go/private:json.bzl", "json_marshal")

"""These are bare-bones Go rules.

In order of priority:

- BUILD file must be written by hand.

- No support for SWIG

- No test sharding or test XML.

"""

_DEFAULT_LIB = "go_default_library"

_VENDOR_PREFIX = "/vendor/"

go_filetype = FileType([
    ".go",
    ".s",
    ".S",
    ".h",  # may be included by .s
])

# be consistent to cc_library.
hdr_exts = [
    ".h",
    ".hh",
    ".hpp",
    ".hxx",
    ".inc",
]

cc_hdr_filetype = FileType(hdr_exts)

# Extensions of files we can build with the Go compiler or with cc_library.
# This is a subset of the extensions recognized by go/build.
cgo_filetype = FileType([
    ".go",
    ".c",
    ".cc",
    ".cxx",
    ".cpp",
    ".s",
    ".S",
    ".h",
    ".hh",
    ".hpp",
    ".hxx",
])

################

def go_environment_vars(ctx):
  """Return a map of environment variables for use with actions, based on
  the arguments. Uses the ctx.fragments.cpp.cpu attribute, if present,
  and picks a default of target_os="linux" and target_arch="amd64"
  otherwise.

  Args:
    The starlark Context.

  Returns:
    A dict of environment variables for running Go tool commands that build for
    the target OS and architecture.
  """
  default_toolchain = {"GOOS": "linux", "GOARCH": "amd64"}
  bazel_to_go_toolchain = {
    "k8": {"GOOS": "linux", "GOARCH": "amd64"},
    "piii": {"GOOS": "linux", "GOARCH": "386"},
    "darwin": {"GOOS": "darwin", "GOARCH": "amd64"},
    "darwin_x86_64": {"GOOS": "darwin", "GOARCH": "amd64"},
    "freebsd": {"GOOS": "freebsd", "GOARCH": "amd64"},
    "armeabi-v7a": {"GOOS": "linux", "GOARCH": "arm"},
    "arm": {"GOOS": "linux", "GOARCH": "arm"}
  }
  env = {}
  if hasattr(ctx.file, "go_tool"):
    env["GOROOT"] = ctx.file.go_tool.dirname + "/.."
  env.update(bazel_to_go_toolchain.get(ctx.fragments.cpp.cpu, default_toolchain))
  return env

def _is_darwin_cpu(ctx):
  cpu = ctx.fragments.cpp.cpu
  return cpu == "darwin" or cpu == "darwin_x86_64"

def _emit_generate_params_action(cmds, ctx, fn):
  cmds_all = [
      # Use bash explicitly. /bin/sh is default, and it may be linked to a
      # different shell, e.g., /bin/dash on Ubuntu.
      "#!/bin/bash",
      "set -e",
  ]
  cmds_all += cmds
  cmds_all_str = "\n".join(cmds_all) + "\n"
  f = ctx.new_file(ctx.configuration.bin_dir, fn)
  ctx.file_action(
      output = f,
      content = cmds_all_str,
      executable = True)
  return f

def _emit_go_asm_action(ctx, source, hdrs, out_obj):
  """Construct the command line for compiling Go Assembly code.
  Constructs a symlink tree to accomodate for workspace name.
  Args:
    ctx: The starlark Context.
    source: a source code artifact
    hdrs: list of .h files that may be included
    out_obj: the artifact (configured target?) that should be produced
  """
  params = {
      "go_tool": ctx.file.go_tool.path,
      "includes": [f.dirname for f in hdrs] + [ctx.file.go_include.path],
      "source": source.path,
      "out": out_obj.path,
  }

  inputs = hdrs + ctx.files.toolchain + [source]
  ctx.action(
      inputs = inputs,
      outputs = [out_obj],
      mnemonic = "GoAsmCompile",
      executable = ctx.executable._asm,
      arguments = [json_marshal(params)],
  )

def _go_importpath(ctx):
  """Returns the expected importpath of the go_library being built.

  Args:
    ctx: The starlark Context

  Returns:
    Go importpath of the library
  """
  path = ctx.attr.importpath
  if path != "":
    return path
  path = ctx.attr.go_prefix.go_prefix
  if path.endswith("/"):
    path = path[:-1]
  if ctx.label.package:
    path += "/" + ctx.label.package
  if ctx.label.name != _DEFAULT_LIB:
    path += "/" + ctx.label.name
  if path.rfind(_VENDOR_PREFIX) != -1:
    path = path[len(_VENDOR_PREFIX) + path.rfind(_VENDOR_PREFIX):]
  if path[0] == "/":
    path = path[1:]
  return path

def _emit_go_compile_action(ctx, sources, deps, libpaths, out_object, gc_goopts):
  """Construct the command line for compiling Go code.

  Args:
    ctx: The starlark Context.
    sources: an iterable of source code artifacts (or CTs? or labels?)
    deps: an iterable of dependencies. Each dependency d should have an
      artifact in d.transitive_go_libraries representing all imported libraries.
    libpaths: the set of paths to search for imported libraries.
    out_object: the object file that should be produced
    gc_goopts: additional flags to pass to the compiler.
  """
  if ctx.coverage_instrumented():
    sources = _emit_go_cover_action(ctx, sources)

  # Compile filtered files.
  args = [
      "-cgo",
      ctx.file.go_tool.path,
      "tool", "compile",
      "-o", out_object.path,
      "-trimpath", "-abs-.",
      "-I", "-abs-.",
  ]
  inputs = depset(sources + ctx.files.toolchain)
  for dep in deps:
    inputs += dep.transitive_go_libraries
  for path in libpaths:
    args += ["-I", path]
  args += gc_goopts + [("" if i.basename.startswith("_cgo") else "-filter-") + i.path for i in sources]
  ctx.action(
      inputs = list(inputs),
      outputs = [out_object],
      mnemonic = "GoCompile",
      executable = ctx.executable._filter_exec,
      arguments = args,
      env = go_environment_vars(ctx),
  )

  return sources

def _emit_go_pack_action(ctx, out_lib, objects):
  """Construct the command line for packing objects together.

  Args:
    ctx: The starlark Context.
    out_lib: the archive that should be produced
    objects: an iterable of object files to be added to the output archive file.
  """
  ctx.action(
      inputs = objects + ctx.files.toolchain,
      outputs = [out_lib],
      mnemonic = "GoPack",
      executable = ctx.file.go_tool,
      arguments = ["tool", "pack", "c", out_lib.path] + [a.path for a in objects],
      env = go_environment_vars(ctx),
  )

def _emit_go_cover_action(ctx, sources):
  """Construct the command line for test coverage instrument.

  Args:
    ctx: The starlark Context.
    sources: an iterable of Go source files.

  Returns:
    A list of Go source code files which might be coverage instrumented.
  """
  outputs = []
  # TODO(linuxerwang): make the mode configurable.
  count = 0

  for src in sources:
    if not src.path.endswith(".go") or src.path.endswith("_test.go"):
      outputs += [src]
      continue

    cover_var = "GoCover_%d" % count
    out = ctx.new_file(src, src.basename[:-3] + '_' + cover_var + '.cover.go')
    outputs += [out]
    ctx.action(
        inputs = [src] + ctx.files.toolchain,
        outputs = [out],
        mnemonic = "GoCover",
        executable = ctx.file.go_tool,
        arguments = ["tool", "cover", "--mode=set", "-var=%s" % cover_var, "-o", out.path, src.path],
        env = go_environment_vars(ctx),
    )
    count += 1

  return outputs

def go_library_impl(ctx):
  """Implements the go_library() rule."""

  sources = depset(ctx.files.srcs)
  go_srcs = depset([s for s in sources if s.basename.endswith('.go')])
  asm_srcs = [s for s in sources if s.basename.endswith('.s') or s.basename.endswith('.S')]
  asm_hdrs = [s for s in sources if s.basename.endswith('.h')]
  deps = ctx.attr.deps
  dep_runfiles = [d.data_runfiles for d in deps]

  cgo_object = None
  if hasattr(ctx.attr, "cgo_object"):
    cgo_object = ctx.attr.cgo_object

  if ctx.attr.library:
    go_srcs += ctx.attr.library.go_sources
    asm_srcs += ctx.attr.library.asm_sources
    asm_hdrs += ctx.attr.library.asm_headers
    deps += ctx.attr.library.direct_deps
    dep_runfiles += [ctx.attr.library.data_runfiles]
    if ctx.attr.library.cgo_object:
      if cgo_object:
        fail("go_library %s cannot have cgo_object because the package " +
             "already has cgo_object in %s" % (ctx.label.name,
                                               ctx.attr.library.name))
      cgo_object = ctx.attr.library.cgo_object
  if not go_srcs:
    fail("may not be empty", "srcs")

  transitive_cgo_deps = depset([], order="topological")
  if cgo_object:
    dep_runfiles += [cgo_object.data_runfiles]
    transitive_cgo_deps += cgo_object.cgo_deps

  extra_objects = [cgo_object.cgo_obj] if cgo_object else []
  for src in asm_srcs:
    obj = ctx.new_file(src, "%s.dir/%s.o" % (ctx.label.name, src.basename[:-2]))
    _emit_go_asm_action(ctx, src, asm_hdrs, obj)
    extra_objects += [obj]

  lib_name = _go_importpath(ctx) + ".a"
  out_lib = ctx.new_file(lib_name)
  out_object = ctx.new_file(ctx.label.name + ".o")
  search_path = out_lib.path[:-len(lib_name)]
  gc_goopts = _gc_goopts(ctx)
  transitive_go_libraries = depset([out_lib])
  transitive_go_library_paths = depset([search_path])
  for dep in deps:
    transitive_go_libraries += dep.transitive_go_libraries
    transitive_cgo_deps += dep.transitive_cgo_deps
    transitive_go_library_paths += dep.transitive_go_library_paths

  go_srcs = _emit_go_compile_action(ctx,
      sources = go_srcs,
      deps = deps,
      libpaths = transitive_go_library_paths,
      out_object = out_object,
      gc_goopts = gc_goopts,
  )
  _emit_go_pack_action(ctx, out_lib, [out_object] + extra_objects)

  dylibs = []
  if cgo_object:
    dylibs += [d for d in cgo_object.cgo_deps if d.path.endswith(".so")]

  runfiles = ctx.runfiles(files = dylibs, collect_data = True)
  for d in dep_runfiles:
    runfiles = runfiles.merge(d)

  return struct(
    label = ctx.label,
    files = depset([out_lib]),
    runfiles = runfiles,
    go_sources = go_srcs,
    asm_sources = asm_srcs,
    asm_headers = asm_hdrs,
    cgo_object = cgo_object,
    direct_deps = ctx.attr.deps,
    transitive_cgo_deps = transitive_cgo_deps,
    transitive_go_libraries = transitive_go_libraries,
    transitive_go_library_paths = transitive_go_library_paths,
    gc_goopts = gc_goopts,
  )

def _c_linker_options(ctx, blacklist=[]):
  """Extracts flags to pass to $(CC) on link from the current context

  Args:
    ctx: the current context
    blacklist: Any flags starts with any of these prefixes are filtered out from
      the return value.

  Returns:
    A list of command line flags
  """
  cpp = ctx.fragments.cpp
  features = ctx.features
  options = cpp.compiler_options(features)
  options += cpp.unfiltered_compiler_options(features)
  options += cpp.link_options
  options += cpp.mostly_static_link_options(ctx.features, False)
  filtered = []
  for opt in options:
    if any([opt.startswith(prefix) for prefix in blacklist]):
      continue
    filtered.append(opt)
  return filtered

def _gc_goopts(ctx):
  gc_goopts = [ctx.expand_make_variables("gc_goopts", f, {})
               for f in ctx.attr.gc_goopts]
  if ctx.attr.library:
    gc_goopts += ctx.attr.library.gc_goopts
  return gc_goopts

def _gc_linkopts(ctx):
  gc_linkopts = [ctx.expand_make_variables("gc_linkopts", f, {})
                 for f in ctx.attr.gc_linkopts]
  for k, v in ctx.attr.x_defs.items():
    gc_linkopts += ["-X", "%s='%s'" % (k, v)]
  return gc_linkopts

def _extract_extldflags(gc_linkopts, extldflags):
  """Extracts -extldflags from gc_linkopts and combines them into a single list.

  Args:
    gc_linkopts: a list of flags passed in through the gc_linkopts attributes.
      ctx.expand_make_variables should have already been applied.
    extldflags: a list of flags to be passed to the external linker.

  Return:
    A tuple containing the filtered gc_linkopts with external flags removed,
    and a combined list of external flags.
  """
  filtered_gc_linkopts = []
  is_extldflags = False
  for opt in gc_linkopts:
    if is_extldflags:
      is_extldflags = False
      extldflags += [opt]
    elif opt == "-extldflags":
      is_extldflags = True
    else:
      filtered_gc_linkopts += [opt]
  return filtered_gc_linkopts, extldflags

def _emit_go_link_action(ctx, transitive_go_library_paths, transitive_go_libraries, cgo_deps, libs,
                         executable, gc_linkopts):
  """Sets up a symlink tree to libraries to link together."""
  config_strip = len(ctx.configuration.bin_dir.path) + 1
  pkg_depth = executable.dirname[config_strip:].count('/') + 1

  ld = "%s" % ctx.fragments.cpp.compiler_executable
  extldflags = _c_linker_options(ctx) + [
      "-Wl,-rpath,$ORIGIN/" + ("../" * pkg_depth),
  ]
  for d in cgo_deps:
    if d.basename.endswith('.so'):
      short_dir = d.dirname[len(d.root.path):]
      extldflags += ["-Wl,-rpath,$ORIGIN/" + ("../" * pkg_depth) + short_dir]
  gc_linkopts, extldflags = _extract_extldflags(gc_linkopts, extldflags)

  link_cmd = [
      ctx.file.go_tool.path,
      "tool", "link",
      "-L", "."
  ]
  for path in transitive_go_library_paths:
    link_cmd += ["-L", path]
  link_cmd += [
      "-o", executable.path,
  ] + gc_linkopts + ['"${STAMP_XDEFS[@]}"']

  # workaround for a bug in ld(1) on Mac OS X.
  # http://lists.apple.com/archives/Darwin-dev/2006/Sep/msg00084.html
  # TODO(yugui) Remove this workaround once rules_go stops supporting XCode 7.2
  # or earlier.
  if not _is_darwin_cpu(ctx):
    link_cmd += ["-s"]

  link_cmd += [
      "-extld", ld,
      "-extldflags", "'%s'" % " ".join(extldflags),
  ] + [lib.path for lib in libs]

  # Avoided -s on OSX but but it requires dsymutil to be on $PATH.
  # TODO(yugui) Remove this workaround once rules_go stops supporting XCode 7.2
  # or earlier.
  cmds = ["export PATH=$PATH:/usr/bin"]

  cmds += [
      "STAMP_XDEFS=()",
  ]

  stamp_inputs = []
  if ctx.attr.linkstamp:
    # read workspace status files, converting "KEY value" lines
    # to "-X $linkstamp.KEY=value" arguments to the go linker.
    stamp_inputs = [ctx.info_file, ctx.version_file]
    for f in stamp_inputs:
      cmds += [
          "while read -r key value || [[ -n $key ]]; do",
          "  STAMP_XDEFS+=(-X \"%s.$key=$value\")" % ctx.attr.linkstamp,
          "done < " + f.path,
      ]

  cmds += [' '.join(link_cmd)]

  f = _emit_generate_params_action(cmds, ctx, lib.basename + ".GoLinkFile.params")

  ctx.action(
      inputs = [f] + (list(transitive_go_libraries) + [lib] + list(cgo_deps) +
                ctx.files.toolchain + ctx.files._crosstool) + stamp_inputs,
      outputs = [executable],
      command = f.path,
      mnemonic = "GoLink",
      env = go_environment_vars(ctx),
  )

def go_binary_impl(ctx):
  """go_binary_impl emits actions for compiling and linking a go executable."""
  lib_result = go_library_impl(ctx)
  _emit_go_link_action(
    ctx,
    transitive_go_libraries=lib_result.transitive_go_libraries,
    transitive_go_library_paths=lib_result.transitive_go_library_paths,
    cgo_deps=lib_result.transitive_cgo_deps,
    libs=lib_result.files,
    executable=ctx.outputs.executable,
    gc_linkopts=_gc_linkopts(ctx))

  return struct(
      files = depset([ctx.outputs.executable]),
      runfiles = lib_result.runfiles,
      cgo_object = lib_result.cgo_object,
  )

def go_test_impl(ctx):
  """go_test_impl implements go testing.

  It emits an action to run the test generator, and then compiles the
  test into a binary."""

  lib_result = go_library_impl(ctx)
  main_go = ctx.new_file(ctx.label.name + "_main_test.go")
  main_object = ctx.new_file(ctx.label.name + "_main_test.o")
  main_lib = ctx.new_file(ctx.label.name + "_main_test.a")
  go_import = _go_importpath(ctx)

  cmds = [
      'UNFILTERED_TEST_FILES=(%s)' %
          ' '.join(["'%s'" % f.path for f in lib_result.go_sources]),
      'FILTERED_TEST_FILES=()',
      'while read -r line; do',
      '  if [ -n "$line" ]; then',
      '    FILTERED_TEST_FILES+=("$line")',
      '  fi',
      'done < <(\'%s\' -cgo "${UNFILTERED_TEST_FILES[@]}")' %
          ctx.executable._filter_tags.path,
      ' '.join([
          "'%s'" % ctx.executable.test_generator.path,
          '--package',
          go_import,
          '--output',
          "'%s'" % main_go.path,
          '"${FILTERED_TEST_FILES[@]}"',
      ]),
  ]
  f = _emit_generate_params_action(
      cmds, ctx, ctx.label.name + ".GoTestGenTest.params")
  inputs = (list(lib_result.go_sources) + list(ctx.files.toolchain) +
            [f, ctx.executable._filter_tags, ctx.executable.test_generator])
  ctx.action(
      inputs = inputs,
      outputs = [main_go],
      command = f.path,
      mnemonic = "GoTestGenTest",
      env = dict(go_environment_vars(ctx), RUNDIR=ctx.label.package))

  _emit_go_compile_action(
    ctx,
    sources=depset([main_go]),
    deps=ctx.attr.deps + [lib_result],
    libpaths=lib_result.transitive_go_library_paths,
    out_object=main_object,
    gc_goopts=_gc_goopts(ctx),
  )
  _emit_go_pack_action(ctx, main_lib, [main_object])
  _emit_go_link_action(
    ctx,
    transitive_go_library_paths=lib_result.transitive_go_library_paths,
    transitive_go_libraries=lib_result.transitive_go_libraries,
    cgo_deps=lib_result.transitive_cgo_deps,
    libs=[main_lib],
    executable=ctx.outputs.executable,
    gc_linkopts=_gc_linkopts(ctx))

  # TODO(bazel-team): the Go tests should do a chdir to the directory
  # holding the data files, so open-source go tests continue to work
  # without code changes.
  runfiles = ctx.runfiles(files = [ctx.outputs.executable])
  runfiles = runfiles.merge(lib_result.runfiles)
  return struct(
      files = depset([ctx.outputs.executable]),
      runfiles = runfiles,
  )

go_env_attrs = {
    "toolchain": attr.label(
        default = Label("//go/toolchain:toolchain"),
        allow_files = True,
        cfg = "host",
    ),
    "go_tool": attr.label(
        default = Label("//go/toolchain:go_tool"),
        single_file = True,
        allow_files = True,
        cfg = "host",
    ),
    "go_prefix": attr.label(
        providers = ["go_prefix"],
        default = Label(
            "//:go_prefix",
            relative_to_caller_repository = True,
        ),
        allow_files = False,
        cfg = "host",
    ),
    "go_src": attr.label(
        default = Label("//go/toolchain:go_src"),
        allow_files = True,
        cfg = "host",
    ),
    "go_include": attr.label(
        default = Label("//go/toolchain:go_include"),
        single_file = True,
        allow_files = True,
        cfg = "host",
    ),
    "go_root": attr.label(
        providers = ["go_root"],
        default = Label(
            "//go/toolchain:go_root",
        ),
        allow_files = False,
        cfg = "host",
    ),
    "_filter_tags": attr.label(
        default = Label("//go/tools/filter_tags"),
        cfg = "host",
        executable = True,
        single_file = True,
    ),
    "_filter_exec": attr.label(
        default = Label("//go/tools/filter_exec"),
        cfg = "host",
        executable = True,
        single_file = True,
    ),
    "_asm": attr.label(
        default = Label("//go/tools/builders:asm"),
        cfg = "host",
        executable = True,
        single_file = True,
    ),
}

go_library_attrs = go_env_attrs + {
    "data": attr.label_list(
        allow_files = True,
        cfg = "data",
    ),
    "srcs": attr.label_list(allow_files = go_filetype),
    "deps": attr.label_list(
        providers = [
            "transitive_go_library_paths",
            "transitive_go_libraries",
            "transitive_cgo_deps",
        ],
    ),
    "importpath": attr.string(),
    "library": attr.label(
        providers = [
            "direct_deps",
            "go_sources",
            "asm_sources",
            "cgo_object",
            "gc_goopts",
        ],
    ),
    "gc_goopts": attr.string_list(),
}

_crosstool_attrs = {
    "_crosstool": attr.label(
        default = Label("//tools/defaults:crosstool"),
    ),
}

go_link_attrs = go_library_attrs + _crosstool_attrs + {
    "gc_linkopts": attr.string_list(),
    "linkstamp": attr.string(),
    "x_defs": attr.string_dict(),
}

go_library = rule(
    go_library_impl,
    attrs = go_library_attrs + {
        "cgo_object": attr.label(
            providers = [
                "cgo_obj",
                "cgo_deps",
            ],
        ),
    },
    fragments = ["cpp"],
)

go_binary = rule(
    go_binary_impl,
    attrs = go_library_attrs + _crosstool_attrs + go_link_attrs,
    executable = True,
    fragments = ["cpp"],
)

go_test = rule(
    go_test_impl,
    attrs = go_library_attrs + _crosstool_attrs + go_link_attrs + {
        "test_generator": attr.label(
            executable = True,
            default = Label(
                "//go/tools:generate_test_main",
            ),
            cfg = "host",
        ),
    },
    executable = True,
    fragments = ["cpp"],
    test = True,
)

def _pkg_dir(workspace_root, package_name):
  if workspace_root and package_name:
    return workspace_root + "/" + package_name
  if workspace_root:
    return workspace_root
  if package_name:
    return package_name
  return "."

def _exec_path(path):
  if path.startswith('/'):
    return path
  return '${execroot}/' + path

def _cgo_filter_srcs_impl(ctx):
  srcs = ctx.files.srcs
  dsts = []
  cmds = []
  for src in srcs:
    stem, _, ext = src.path.rpartition('.')
    dst_basename = "%s.filtered.%s" % (stem, ext)
    dst = ctx.new_file(src, dst_basename)
    cmds += [
        "if '%s' -cgo -quiet '%s'; then" %
            (ctx.executable._filter_tags.path, src.path),
        "  cp '%s' '%s'" % (src.path, dst.path),
        "else",
        "  echo -n >'%s'" % dst.path,
        "fi",
    ]
    dsts.append(dst)

  if ctx.label.package == "":
    script_name = ctx.label.name + ".CGoFilterSrcs.params"
  else:
    script_name = ctx.label.package + "/" + ctx.label.name + ".CGoFilterSrcs.params"
  f = _emit_generate_params_action(cmds, ctx, script_name)
  ctx.action(
      inputs = [f, ctx.executable._filter_tags] + srcs,
      outputs = dsts,
      command = f.path,
      mnemonic = "CgoFilterSrcs",
  )
  return struct(
      files = depset(dsts),
  )

_cgo_filter_srcs = rule(
    implementation = _cgo_filter_srcs_impl,
    attrs = {
        "srcs": attr.label_list(
            allow_files = cgo_filetype,
        ),
        "_filter_tags": attr.label(
            default = Label("//go/tools/filter_tags"),
            cfg = "host",
            executable = True,
            single_file = True,
        ),
    },
    fragments = ["cpp"],
)

def _cgo_codegen_impl(ctx):
  go_srcs = ctx.files.srcs
  srcs = go_srcs + ctx.files.c_hdrs
  linkopts = ctx.attr.linkopts
  copts = ctx.fragments.cpp.c_options + ctx.attr.copts
  deps = depset([], order="topological")
  for d in ctx.attr.deps:
    srcs += list(d.cc.transitive_headers)
    deps += d.cc.libs
    copts += ['-D' + define for define in d.cc.defines]
    for inc in d.cc.include_directories:
      copts += ['-I', _exec_path(inc)]
    for hdr in ctx.files.c_hdrs:
        copts += ['-iquote', hdr.dirname]
    for inc in d.cc.quote_include_directories:
      copts += ['-iquote', _exec_path(inc)]
    for inc in d.cc.system_include_directories:
      copts += ['-isystem',  _exec_path(inc)]
    for lib in d.cc.libs:
      if lib.basename.startswith('lib') and lib.basename.endswith('.so'):
        linkopts += ['-L', lib.dirname, '-l', lib.basename[3:-3]]
      else:
        linkopts += [lib.path]
    linkopts += d.cc.link_flags

  p = _pkg_dir(ctx.label.workspace_root, ctx.label.package) + "/"
  if p == "./":
    p = "" # workaround when cgo_library in repository root
  out_dir = (ctx.configuration.genfiles_dir.path + '/' +
             p + ctx.attr.outdir)
  cc = ctx.fragments.cpp.compiler_executable
  cmds = [
      # We cannot use env for CC because $(CC) on OSX is relative
      # and '../' does not work fine due to symlinks.
      'export CC=$(cd $(dirname {cc}); pwd)/$(basename {cc})'.format(cc=cc),
      'export CXX=$CC',
      'objdir="%s/gen"' % out_dir,
      'execroot=$(pwd)',
      'mkdir -p "$objdir"',
      'unfiltered_go_files=(%s)' % ' '.join(["'%s'" % f.path for f in go_srcs]),
      'filtered_go_files=()',
      'for file in "${unfiltered_go_files[@]}"; do',
      '  stem=$(basename "$file" .go)',
      '  if %s -cgo -quiet "$file"; then' % ctx.executable._filter_tags.path,
      '    filtered_go_files+=("$file")',
      '  else',
      '    grep --max-count 1 "^package " "$file" >"$objdir/$stem.go"',
      '    echo -n >"$objdir/$stem.c"',
      '  fi',
      'done',
      'if [ ${#filtered_go_files[@]} -eq 0 ]; then',
      '  echo no buildable Go source files in %s >&1' % str(ctx.label),
      '  exit 1',
      'fi',
      '"$GOROOT/bin/go" tool cgo -objdir "$objdir" -- %s "${filtered_go_files[@]}"' %
          ' '.join(['"%s"' % copt for copt in copts]),
      # Rename the outputs using glob so we don't have to understand cgo's mangling
      # TODO(#350): might be fixed by this?.
      'for file in "${filtered_go_files[@]}"; do',
      '  stem=$(basename "$file" .go)',
      '  mv "$objdir/"*"$stem.cgo1.go" "$objdir/$stem.go"',
      '  mv "$objdir/"*"$stem.cgo2.c" "$objdir/$stem.c"',
      'done',
      'rm -f $objdir/_cgo_.o $objdir/_cgo_flags',
    ]

  f = _emit_generate_params_action(cmds, ctx, out_dir + ".CGoCodeGenFile.params")

  inputs = (srcs + ctx.files.toolchain + ctx.files._crosstool +
            [f, ctx.executable._filter_tags])
  ctx.action(
      inputs = inputs,
      outputs = ctx.outputs.outs,
      mnemonic = "CGoCodeGen",
      progress_message = "CGoCodeGen %s" % ctx.label,
      command = f.path,
      env = go_environment_vars(ctx) + {
          "CGO_LDFLAGS": " ".join(linkopts),
      },
  )
  return struct(
      label = ctx.label,
      files = depset(ctx.outputs.outs),
      cgo_deps = deps,
  )

_cgo_codegen_rule = rule(
    _cgo_codegen_impl,
    attrs = go_env_attrs + _crosstool_attrs + {
        "srcs": attr.label_list(
            allow_files = go_filetype,
            non_empty = True,
        ),
        "c_hdrs": attr.label_list(
            allow_files = cc_hdr_filetype,
        ),
        "deps": attr.label_list(
            allow_files = False,
            providers = ["cc"],
        ),
        "copts": attr.string_list(),
        "linkopts": attr.string_list(),
        "outdir": attr.string(mandatory = True),
        "outs": attr.output_list(
            mandatory = True,
            non_empty = True,
        ),
    },
    fragments = ["cpp"],
    output_to_genfiles = True,
)

def _cgo_codegen(name, srcs, c_hdrs=[], deps=[], copts=[], linkopts=[],
                 go_tool=None, toolchain=None):
  """Generates glue codes for interop between C and Go

  Args:
    name: A unique name of the rule
    srcs: list of Go source files.
      Each of them must contain `import "C"`.
    c_hdrs: C/C++ header files necessary to determine kinds of
      C/C++ identifiers in srcs.
    deps: A list of cc_library rules.
      The generated codes are expected to be linked with these deps.
    linkopts: A list of linker options,
      These flags are passed to the linker when the generated codes
      are linked into the target binary.
  """
  outdir = name + ".dir"
  outgen = outdir + "/gen"

  go_thunks = []
  c_thunks = []
  for s in srcs:
    if not s.endswith('.go'):
      fail("not a .go file: %s" % s)
    basename = s[:-3]
    if basename.rfind("/") >= 0:
      basename = basename[basename.rfind("/")+1:]
    go_thunks.append(outgen + "/" + basename + ".go")
    c_thunks.append(outgen + "/" + basename + ".c")

  outs = struct(
      name = name,

      outdir = outgen,
      go_thunks = go_thunks,
      c_thunks = c_thunks,
      c_exports = [
          outgen + "/_cgo_export.c",
          outgen + "/_cgo_export.h",
      ],
      c_dummy = outgen + "/_cgo_main.c",
      gotypes = outgen + "/_cgo_gotypes.go",
  )

  _cgo_codegen_rule(
      name = name,
      srcs = srcs,
      c_hdrs = c_hdrs,
      deps = deps,
      copts = copts,
      linkopts = linkopts,

      go_tool = go_tool,
      toolchain = toolchain,

      outdir = outdir,
      outs = outs.go_thunks + outs.c_thunks + outs.c_exports + [
          outs.c_dummy, outs.gotypes,
      ],

      visibility = ["//visibility:private"],
  )
  return outs

def _cgo_import_impl(ctx):
  cmds = [
      (ctx.file.go_tool.path + " tool cgo" +
       " -dynout " + ctx.outputs.out.path +
       " -dynimport " + ctx.file.cgo_o.path +
       " -dynpackage $(%s %s)"  % (ctx.executable._extract_package.path,
                                   ctx.file.sample_go_src.path)),
  ]
  f = _emit_generate_params_action(cmds, ctx, ctx.outputs.out.path + ".CGoImportGenFile.params")
  ctx.action(
      inputs = (ctx.files.toolchain +
                [f, ctx.file.go_tool, ctx.executable._extract_package,
                 ctx.file.cgo_o, ctx.file.sample_go_src]),
      outputs = [ctx.outputs.out],
      command = f.path,
      mnemonic = "CGoImportGen",
      env = go_environment_vars(ctx),
  )
  return struct(
      files = depset([ctx.outputs.out]),
  )

_cgo_import = rule(
    _cgo_import_impl,
    attrs = go_env_attrs + {
        "cgo_o": attr.label(
            allow_files = True,
            single_file = True,
        ),
        "sample_go_src": attr.label(
            allow_files = True,
            single_file = True,
        ),
        "out": attr.output(
            mandatory = True,
        ),
        "_extract_package": attr.label(
            default = Label("//go/tools/extract_package"),
            executable = True,
            cfg = "host",
        ),
    },
    fragments = ["cpp"],
)

def _cgo_genrule_impl(ctx):
  return struct(
    label = ctx.label,
    go_sources = ctx.files.srcs,
    asm_sources = [],
    asm_headers = [],
    cgo_object = ctx.attr.cgo_object,
    direct_deps = ctx.attr.deps,
    gc_goopts = [],
  )

_cgo_genrule = rule(
    _cgo_genrule_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = FileType([".go"])),
        "cgo_object": attr.label(
            providers = [
                "cgo_obj",
                "cgo_deps",
            ],
        ),
        "deps": attr.label_list(
            providers = [
                "direct_deps",
                "transitive_go_library_paths",
                "transitive_go_libraries",
                "transitive_cgo_deps",
            ],
        ),
    },
    fragments = ["cpp"],
)

"""Generates symbol-import directives for cgo

Args:
  cgo_o: The loadable object to extract dynamic symbols from.
  sample_go_src: A go source which is compiled together with the generated file.
    The generated file will have the same Go package name as this file.
  out: Destination of the generated codes.
"""

def _cgo_object_impl(ctx):
  arguments = _c_linker_options(ctx, blacklist=[
      # never link any dependency libraries
      "-l", "-L",
      # manage flags to ld(1) by ourselves
      "-Wl,"])
  arguments += [
      "-o", ctx.outputs.out.path,
      "-nostdlib",
      "-Wl,-r",
  ]
  if _is_darwin_cpu(ctx):
    arguments += ["-shared", "-Wl,-all_load"]
  else:
    arguments += ["-Wl,-whole-archive"]

  lo = ctx.files.src[-1]
  arguments += [lo.path]

  ctx.action(
      inputs = [lo] + ctx.files._crosstool,
      outputs = [ctx.outputs.out],
      mnemonic = "CGoObject",
      progress_message = "Linking %s" % ctx.outputs.out.short_path,
      executable = ctx.fragments.cpp.compiler_executable,
      arguments = arguments,
  )
  runfiles = ctx.runfiles(collect_data = True)
  runfiles = runfiles.merge(ctx.attr.src.data_runfiles)
  return struct(
      files = depset([ctx.outputs.out]),
      cgo_obj = ctx.outputs.out,
      cgo_deps = ctx.attr.cgogen.cgo_deps,
      runfiles = runfiles,
  )

_cgo_object = rule(
    _cgo_object_impl,
    attrs = _crosstool_attrs + {
        "src": attr.label(
            mandatory = True,
            providers = ["cc"],
        ),
        "cgogen": attr.label(
            mandatory = True,
            providers = ["cgo_deps"],
        ),
        "out": attr.output(
            mandatory = True,
        ),
    },
    fragments = ["cpp"],
)

"""Generates _all.o to be archived together with Go objects.

Args:
  src: source static library which contains objects
  cgogen: _cgo_codegen rule which knows the dependency cc_library() rules
    to be linked together with src when we generate the final go binary.
"""

def _setup_cgo_library(name, srcs, cdeps, copts, clinkopts, go_tool, toolchain):
  go_srcs = [s for s in srcs if s.endswith('.go')]
  c_hdrs = [s for s in srcs if any([s.endswith(ext) for ext in hdr_exts])]
  c_srcs = [s for s in srcs if not s in (go_srcs + c_hdrs)]

  # Split cgo files into .go parts and .c parts (plus some other files).
  cgogen = _cgo_codegen(
      name = name + ".cgo",
      srcs = go_srcs,
      c_hdrs = c_hdrs,
      deps = cdeps,
      copts = copts,
      linkopts = clinkopts,
      go_tool = go_tool,
      toolchain = toolchain,
  )

  # Filter c_srcs with build constraints.
  c_filtered_srcs = []
  if len(c_srcs) > 0:
    c_filtered_srcs_name = name + "_filter_cgo_srcs"
    _cgo_filter_srcs(
        name = c_filtered_srcs_name,
        srcs = c_srcs,
    )
    c_filtered_srcs.append(":" + c_filtered_srcs_name)

  pkg_dir = _pkg_dir(
      "external/" + REPOSITORY_NAME[1:] if len(REPOSITORY_NAME) > 1 else "",
      PACKAGE_NAME)

  # Platform-specific settings
  native.config_setting(
      name = name + "_windows_setting",
      values = {
          "cpu": "x64_windows_msvc",
      },
  )
  platform_copts = select({
      ":" + name + "_windows_setting": ["-mthreads"],
      "//conditions:default": ["-pthread"],
  })
  platform_linkopts = select({
      ":" + name + "_windows_setting": ["-mthreads"],
      "//conditions:default": ["-pthread"],
  })

  # Bundles objects into an archive so that _cgo_.o and _all.o can share them.
  native.cc_library(
      name = cgogen.outdir + "/_cgo_lib",
      srcs = cgogen.c_thunks + cgogen.c_exports + c_filtered_srcs + c_hdrs,
      deps = cdeps,
      copts = copts + platform_copts + [
          "-I", pkg_dir,
          "-I", "$(GENDIR)/" + pkg_dir + "/" + cgogen.outdir,
          # The generated thunks often contain unused variables.
          "-Wno-unused-variable",
      ],
      linkopts = clinkopts + platform_linkopts,
      linkstatic = 1,
      # _cgo_.o and _all.o keep all objects in this archive.
      # But it should not be very annoying in the final binary target
      # because _cgo_object rule does not propagate alwayslink=1
      alwayslink = 1,
      visibility = ["//visibility:private"],
  )

  # Loadable object which cgo reads when it generates _cgo_import.go
  native.cc_binary(
      name = cgogen.outdir + "/_cgo_.o",
      srcs = [cgogen.c_dummy],
      deps = cdeps + [cgogen.outdir + "/_cgo_lib"],
      copts = copts,
      linkopts = clinkopts,
      visibility = ["//visibility:private"],
  )
  _cgo_import(
      name = "%s.cgo.importgen" % name,
      cgo_o = cgogen.outdir + "/_cgo_.o",
      out = cgogen.outdir + "/_cgo_import.go",
      sample_go_src = go_srcs[0],
      go_tool = go_tool,
      toolchain = toolchain,
      visibility = ["//visibility:private"],
  )

  _cgo_object(
      name = cgogen.outdir + "/_cgo_object",
      src = cgogen.outdir + "/_cgo_lib",
      out = cgogen.outdir + "/_all.o",
      cgogen = cgogen.name,
      visibility = ["//visibility:private"],
  )
  return cgogen

def cgo_genrule(name, srcs,
                copts=[],
                clinkopts=[],
                cdeps=[],
                **kwargs):
  cgogen = _setup_cgo_library(
      name = name,
      srcs = srcs,
      cdeps = cdeps,
      copts = copts,
      clinkopts = clinkopts,
      toolchain = None,
      go_tool = None,
  )
  _cgo_genrule(
      name = name,
      srcs = cgogen.go_thunks + [
          cgogen.gotypes,
          cgogen.outdir + "/_cgo_import.go",
      ],
      cgo_object = cgogen.outdir + "/_cgo_object",
      **kwargs
  )

def cgo_library(name, srcs,
                toolchain=None,
                go_tool=None,
                copts=[],
                clinkopts=[],
                cdeps=[],
                **kwargs):
  """Builds a cgo-enabled go library.

  Args:
    name: A unique name for this rule.
    srcs: List of Go, C and C++ files that are processed to build a Go library.
      Those Go files must contain `import "C"`.
      C and C++ files can be anything allowed in `srcs` attribute of
      `cc_library`.
    copts: Add these flags to the C++ compiler.
    clinkopts: Add these flags to the C++ linker.
    cdeps: List of C/C++ libraries to be linked into the binary target.
      They must be `cc_library` rules.
    deps: List of other libraries to be linked to this library target.
    data: List of files needed by this rule at runtime.

  NOTE:
    `srcs` cannot contain pure-Go files, which do not have `import "C"`.
    So you need to define another `go_library` when you build a go package with
    both cgo-enabled and pure-Go sources.

    ```
    cgo_library(
        name = "cgo_enabled",
        srcs = ["cgo-enabled.go", "foo.cc", "bar.S", "baz.a"],
    )

    go_library(
        name = "go_default_library",
        srcs = ["pure-go.go"],
        library = ":cgo_enabled",
    )
    ```
  """
  cgogen = _setup_cgo_library(
      name = name,
      srcs = srcs,
      cdeps = cdeps,
      copts = copts,
      clinkopts = clinkopts,
      go_tool = go_tool,
      toolchain = toolchain,
  )

  go_library(
      name = name,
      srcs = cgogen.go_thunks + [
          cgogen.gotypes,
          cgogen.outdir + "/_cgo_import.go",
      ],
      cgo_object = cgogen.outdir + "/_cgo_object",
      go_tool = go_tool,
      toolchain = toolchain,
      **kwargs
  )
