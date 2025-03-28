"""Rules for packaging Windows binaries in ZIP format."""

load("@rules_pkg//pkg:mappings.bzl", "pkg_attributes", "pkg_filegroup", "pkg_files")
load("@rules_pkg//pkg/private/zip:zip.bzl", "pkg_zip")

def windows_bin_zip(name, binary_target, dev_version = "0.0.0", visibility = None):
    """Creates a ZIP archive for a Windows binary using rules_pkg.

    This rule packages a Windows binary into a ZIP file suitable for distribution.
    By default it uses the provided dev_version ("0.0.0") for development builds.
    In release workflow, the file can be processed to replace this version with the actual release version.

    Args:
        name: Name for the zip target
        binary_target: The binary target to package
        dev_version: Default version to use for development builds (defaults to "0.0.0")
        visibility: Visibility specification for the generated targets
    """

    # Extract arch from target name (assuming format "garf-bin-windows-amd64" or similar)
    arch = binary_target.split("-")[-1]

    # Copy the binary to a predictable name for renaming
    # Use platform-specific commands to avoid dependency on bash on Windows
    native.genrule(
        name = name + "_renamed_binary",
        srcs = [binary_target],
        outs = [name + "_renamed.exe"],
        # Platform-specific commands to copy the binary
        cmd = "cp $(location %s) $@" % binary_target,  # Default fallback
        cmd_bash = "cp $(location %s) $@" % binary_target,  # Unix/Linux/macOS
        cmd_bat = "copy $(location %s) $@" % binary_target,  # Windows cmd.exe
        cmd_ps = "Copy-Item -Path $(location %s) -Destination $@" % binary_target,  # Windows PowerShell
        executable = False,
    )

    # Create mapping for the renamed file
    pkg_files(
        name = name + "_files",
        srcs = [":" + name + "_renamed_binary"],
        attributes = pkg_attributes(
            mode = "0755",  # Executable permission
        ),
        renames = {
            name + "_renamed.exe": "garf-" + dev_version + "-windows-" + arch + ".exe",
        },
    )

    # Group all files for the package
    pkg_filegroup(
        name = name + "_pkg_files",
        srcs = [":" + name + "_files"],
    )

    # Create the final ZIP package - use a clean output name
    pkg_zip(
        name = name,
        srcs = [":" + name + "_pkg_files"],
        out = name.replace("-zip", "") + ".zip",  # Remove "-zip" from the output filename
        compression_type = "deflated",
        compression_level = 9,
        visibility = visibility or ["//visibility:public"],
    )

def windows_bin_zips(name, base_name = None, arches = None, dev_version = "0.0.0", visibility = None):
    """Creates ZIP archives for Windows binaries across multiple architectures.

    This is a convenience wrapper around windows_bin_zip that creates ZIP packages
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

    for arch in arches:
        windows_bin_zip(
            name = base_name + "-windows-" + arch + "-zip",
            binary_target = ":" + base_name + "-windows-" + arch,
            dev_version = dev_version,
            visibility = visibility,
        )
