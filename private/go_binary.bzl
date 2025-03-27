"""Macros for building Go binaries across multiple platforms."""

load("@rules_go//go:def.bzl", "go_binary")

def cross_platform_go_binary(name, embed, visibility = None):
    """Creates Go binaries for multiple platforms simultaneously.

    This macro creates separate go_binary targets for each platform and architecture
    combination, allowing for cross-platform builds in a single Bazel invocation.

    Args:
        name: Base name for the binaries. Will be extended with OS/arch suffixes.
        embed: Library target(s) to embed in the binary.
        visibility: Visibility specification for the generated targets.
            Defaults to public visibility if None.

    Example:
        cross_platform_go_binary(
            name = "myapp",
            embed = [":myapp_lib"],
        )
    """
    for os in ("linux", "darwin", "windows"):
        ext = ".exe" if os == "windows" else ""

        # Don't strip debugging symbols on Windows, as it makes binaries more
        # likely to be flagged as malware.
        gc_linkopts = [] if os == "windows" else ["-s", "-w"]

        for arch in ("amd64", "arm64"):
            go_binary(
                name = "%s-%s-%s" % (name, os, arch),
                out = "%s_%s_%s%s" % (name, os, arch, ext),
                embed = embed,
                gc_linkopts = gc_linkopts,
                goarch = arch,
                goos = os,
                pure = "on",
                visibility = visibility or ["//visibility:public"],
            )
