ifeq ($(OS),Windows_NT)
  export BAZEL_SH=C:\msys64\usr\bin\bash.exe
endif

build:
	bazelisk build //:garf
.PHONY: build

fmt:
	bazelisk run @rules_go//go -- fmt ./...
.PHONY: fmt

vet:
	bazelisk run @rules_go//go -- vet ./...
.PHONY: vet

# go mod tidy is currently WIP
# https://github.com/bazelbuild/bazel-gazelle/pull/1495
tidy:
	bazelisk run @rules_go//go -- mod tidy -v
.PHONY: tidy

gazelle:
	bazelisk run //:gazelle
.PHONY: gazelle

go-get:
	bazelisk run @rules_go//go -- get -u $(DEP)
.PHONY: go-get
