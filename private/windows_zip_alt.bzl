"""Alternative implementation of Windows ZIP packaging using aspect_bazel_lib's bsdtar."""

load("@aspect_bazel_lib//lib:tar.bzl", "tar")

# Constants for reuse
_EXE_EXTENSION = ".exe"
_ZIP_EXTENSION = ".zip"
_ALT_SUFFIX = "_alt"
_DEFAULT_VERSION = "0.0.0"
_DEFAULT_ARCHES = ["amd64", "arm64"]

def _extract_arch(binary_target):  # type: (str) -> str
    """Extracts architecture from binary target name.
    
    Args:
        binary_target: String, the binary target name
        
    Returns:
        String, the extracted architecture
    """
    parts = binary_target.split("-")
    if len(parts) < 2:
        fail("Binary target name must contain architecture: {}".format(binary_target))
    return parts[-1]

def windows_bin_zip_alt(name, binary_target, dev_version = _DEFAULT_VERSION, visibility = None):  # type: (str, str, str, list[str] | None) -> str
    """Creates a ZIP archive for a Windows binary using aspect_bazel_lib's bsdtar.

    This implementation uses bsdtar's ability to create ZIP files directly by
    specifying the --format=zip flag.

    Args:
        name: String, name for the zip target
        binary_target: String, the binary target to package
        dev_version: String, version to use for development builds (defaults to "0.0.0")
        visibility: List of labels, visibility specification for the generated targets
        
    Returns:
        String, the name of the created ZIP target with suffix
        
    Example:
        windows_bin_zip_alt(
            name = "app-windows-amd64-zip",
            binary_target = ":app-windows-amd64",
            dev_version = "1.0.0",
        )
    """
    # Extract arch from target name (assuming format "garf-bin-windows-amd64" or similar)
    arch = _extract_arch(binary_target)
    
    # Use a different name to avoid conflicts with the existing rule
    alt_name = name + _ALT_SUFFIX
    zip_output = alt_name.replace("-zip" + _ALT_SUFFIX, "") + _ZIP_EXTENSION
    
    # Create a properly named copy of the binary
    versioned_binary = alt_name + "_renamed"
    versioned_binary_out = "garf-" + dev_version + "-windows-" + arch + _EXE_EXTENSION
    
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
    
    return alt_name

def windows_bin_zips_alt(name, base_name = None, arches = None, dev_version = _DEFAULT_VERSION, visibility = None):  # type: (str, str | None, list[str] | None, str, list[str] | None) -> None
    """Creates ZIP archives for Windows binaries across multiple architectures.
    
    This is a convenience wrapper that creates ZIP packages
    for all specified Windows architectures with a single function call.
    
    Args:
        name: String, a unique name for this target (required by Bazel convention)
        base_name: String, base name for binaries (e.g., "garf-bin"). Defaults to name.
        arches: List of strings, architectures to create ZIPs for (defaults to ["amd64", "arm64"])
        dev_version: String, version to use for development builds (defaults to "0.0.0")
        visibility: List of labels, visibility specification for the generated targets
    
    Example:
        windows_bin_zips_alt(
            name = "windows-zips",
            base_name = "app-bin",
            arches = ["amd64", "arm64", "386"],
            dev_version = "1.2.3",
        )
    """
    if base_name == None:
        base_name = name
        
    if arches == None:
        arches = _DEFAULT_ARCHES
        
    # Create a list of all individual zip targets
    zip_targets = []
    
    for arch in arches:
        target_name = base_name + "-windows-" + arch + "-zip"
        alt_name = windows_bin_zip_alt(
            name = target_name,
            binary_target = ":" + base_name + "-windows-" + arch,
            dev_version = dev_version,
            visibility = visibility,
        )
        zip_targets.append(":" + alt_name)
    
    # Create an empty file to depend on all zip targets
    native.genrule(
        name = name,
        srcs = zip_targets,
        outs = [name + ".done"],
        cmd = "touch $@",
        visibility = visibility,
    )
