"""Macros for building Go binaries across multiple platforms."""

load("@rules_go//go:def.bzl", "go_binary")

def cross_platform_go_binary(name, embed, visibility = None, x_defs = None, tags = None):
    # type: (string, list[string], list[string] | None, dict[string, string] | None, list[string] | None) -> None
    """Creates Go binaries for multiple platforms simultaneously.

    This macro creates separate go_binary targets for each platform and architecture
    combination, allowing for cross-platform builds in a single Bazel invocation.

    Args:
        name: Base name for the binaries. Will be extended with OS/arch suffixes.
        embed: Library target(s) to embed in the binary.
        visibility: Visibility specification for the generated targets.
            Defaults to public visibility if None.
        x_defs: Map of string to string for version stamping.
            Keys are variable names to substitute, values are the strings to use.
        tags: List of tags to apply to the generated targets.

    Example:
        cross_platform_go_binary(
            name = "myapp",
            embed = [":myapp_lib"],
            x_defs = {"Version": "1.0.0"},
            tags = ["manual"],
        )
    """
    for os in ("linux", "darwin", "windows"):  # type: string
        ext = ".exe" if os == "windows" else ""

        # Don't strip debugging symbols on Windows, as it makes binaries more
        # likely to be flagged as malware.
        gc_linkopts = [] if os == "windows" else ["-s", "-w"]

        for arch in ("amd64", "arm64"):  # type: string
            go_binary(
                name = "%s-%s-%s" % (name, os, arch),
                out = "%s_%s_%s%s" % (name, os, arch, ext),
                embed = embed,
                gc_linkopts = gc_linkopts,
                goarch = arch,
                goos = os,
                pure = "on",
                visibility = visibility or ["//visibility:public"],
                x_defs = x_defs,
                tags = tags,
            )
