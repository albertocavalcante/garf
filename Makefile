build:
	bazel build //src:cli
.PHONY: build

fmt:
	bazel run @rules_go//go -- fmt ./...
.PHONY: fmt

vet:
	bazel run @rules_go//go -- vet ./...
.PHONY: vet

# go mod tidy is currently WIP
# https://github.com/bazelbuild/bazel-gazelle/pull/1495
tidy:
	bazel run @rules_go//go -- mod tidy -v
.PHONY: tidy

gazelle:
	bazel run //:gazelle
.PHONY: gazelle

go-get:
	bazel run @rules_go//go -- get -u $(DEP)
.PHONY: go-get
