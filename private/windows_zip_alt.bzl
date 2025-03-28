"""Alternative implementation of Windows ZIP packaging using aspect_bazel_lib's bsdtar."""

load("@aspect_bazel_lib//lib:tar.bzl", "tar")

def windows_bin_zip_alt(name, binary_target, dev_version = "0.0.0", visibility = None):
    """Creates a ZIP archive for a Windows binary using aspect_bazel_lib's bsdtar.

    This implementation uses bsdtar's ability to create ZIP files directly by
    specifying the --format=zip flag.

    Args:
        name: Name for the zip target
        binary_target: The binary target to package
        dev_version: Default version to use for development builds (defaults to "0.0.0")
        visibility: Visibility specification for the generated targets
    """
    # Extract arch from target name (assuming format "garf-bin-windows-amd64" or similar)
    arch = binary_target.split("-")[-1]
    
    # Use a different name to avoid conflicts with the existing rule
    alt_name = name + "_alt"
    zip_output = alt_name.replace("-zip-alt", "") + ".zip"
    
    # Create a properly named copy of the binary
    versioned_binary = alt_name + "_renamed"
    versioned_binary_out = "garf-" + dev_version + "-windows-" + arch + ".exe"
    
    native.genrule(
        name = versioned_binary,
        srcs = [binary_target],
        outs = [versioned_binary_out],
        cmd = select({
            "@platforms//os:windows": "copy $(location %s) $@" % binary_target,
            "//conditions:default": "cp $(location %s) $@" % binary_target,
        }),
    )
    
    # Create the ZIP archive directly using aspect_bazel_lib's tar rule with --format=zip
    tar(
        name = alt_name,
        srcs = [":" + versioned_binary],
        out = zip_output,
        args = ["--format=zip"],
        visibility = visibility or ["//visibility:public"],
    )

def windows_bin_zips_alt(name, base_name = None, arches = None, dev_version = "0.0.0", visibility = None):
    """Creates ZIP archives for Windows binaries across multiple architectures.
    
    This is a convenience wrapper that creates ZIP packages
    for all specified Windows architectures with a single function call.
    
    Args:
        name: A unique name for this target (required by Bazel convention)
        base_name: Base name for binaries (e.g., "garf-bin"). Defaults to name if not provided.
        arches: List of architectures to create ZIPs for (defaults to ["amd64", "arm64"])
        dev_version: Default version to use for development builds (defaults to "0.0.0")
        visibility: Visibility specification for the generated targets
    """
    if base_name == None:
        base_name = name
        
    if arches == None:
        arches = ["amd64", "arm64"]
        
    # Create a list of all individual zip targets
    zip_targets = []
    
    for arch in arches:
        target_name = base_name + "-windows-" + arch + "-zip"
        windows_bin_zip_alt(
            name = target_name,
            binary_target = ":" + base_name + "-windows-" + arch,
            dev_version = dev_version,
            visibility = visibility,
        )
        zip_targets.append(":" + target_name + "_alt")
    
    # Create an empty file to depend on all zip targets
    native.genrule(
        name = name,
        srcs = zip_targets,
        outs = [name + ".done"],
        cmd = "touch $@",
        visibility = visibility,
    )
