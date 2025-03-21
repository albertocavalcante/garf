"""Nogo configuration for the project."""

load("@rules_go//go:def.bzl", "nogo")
load("//tools/nogo:analyzers.bzl", "complete_analyzers_suite")

def setup_nogo(name = "garf_nogo"):
    """Creates and registers the nogo target.

    Args:
        name: The name for the nogo target (defaults to "garf_nogo").

    Returns:
        None.
    """
    nogo(
        name = name,
        visibility = ["//visibility:public"],
        deps = complete_analyzers_suite(),
        config = "//tools/nogo:config.json",
    )
