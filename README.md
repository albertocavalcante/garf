# CLI

I don't know what is CLI will be yet, for now it will serve just as a sample boilerplate

## Running

```sh
bazel run //src:cli
```

## Build

### Bazel

This uses Bazel as its build tool.
It's a go project, which relies on [rules_go](https://registry.bazel.build/modules/rules_go), [gazelle](https://registry.bazel.build/modules/gazelle) and [bzlmod](https://bazel.build/external/overview#bzlmod).

**Reference**
- [Go with Bzlmod](https://github.com/bazelbuild/rules_go/blob/master/docs/go/core/bzlmod.md)

#### Bazel Tools

##### Buildifier

- [Documentation](https://github.com/bazelbuild/buildtools/blob/master/buildifier/README.md)

```sh
go install github.com/bazelbuild/buildtools/buildifier@latest
```

##### Buildozer

- [Documentation](https://github.com/bazelbuild/buildtools/blob/master/buildozer/README.md)

```sh
go install github.com/bazelbuild/buildtools/buildozer@latest
```

**Sample Usage**

```sh
buildozer 'use_repo_remove @gazelle//:extensions.bzl go_deps com_github_spf13_cobra' //MODULE.bazel:all
```

#### Adding third party dependencies

For example, in order to add cobra, run: 
```sh
make go-get DEP=github.com/spf13/cobra@latest
```
