"""Nogo dependency labels for Go analyzers."""

def generate_analyzers_labels(base_path, items):
    # type: (string, list) -> list
    """Generates labels by prefixing each item with a base path.

    Args:
        base_path: The common prefix for all labels.
        items: A list of items to be converted into labels.

    Returns:
        A list of generated labels.
    """
    return [base_path + item for item in items]

def standard_go_analyzers_labels():
    # type: () -> list
    """Generates nogo dependency labels for standard Go analyzers.

    Source: 
        Vet Analyzers: https://cs.opensource.google/go/go/+/refs/tags/go1.22.5:src/cmd/vet/main.go;l=12-43

    Returns:
        A list of dependency labels for standard Go analyzers.
    """

    analyzers = [
        "appends",
        "asmdecl",
        "assign",
        "atomic",
        "bools",
        "buildtag",
        "cgocall",
        "composite",
        "copylock",
        "defers",
        "directive",
        "errorsas",
        "framepointer",
        "httpresponse",
        "ifaceassert",
        "loopclosure",
        "lostcancel",
        "nilfunc",
        "printf",
        "shift",
        "sigchanyzer",
        "slog",
        "stdmethods",
        "stringintconv",
        "structtag",
        "testinggoroutine",
        "tests",
        "timeformat",
        "unmarshal",
        "unreachable",
        "unsafeptr",
        "unusedresult",
    ]

    # https://pkg.go.dev/golang.org/x/tools/go/analysis/passes
    base_path = "@org_golang_x_tools//go/analysis/passes/"
    return generate_analyzers_labels(base_path, analyzers)

def extended_analyzers_labels():
    # type: () -> list[Label]
    """Returns a list of extended Go analyzer labels.
    
    These are additional analyzers beyond the standard Go toolchain
    that provide extra checks and validations.
    
    Returns:
        A list of extended analyzer labels.
    """
    return []

def complete_analyzers_suite():
    # type: () -> list
    """Generates a comprehensive list of all Go analyzers for nogo.
    
    Combines standard Go analyzers with extended analyzers
    to provide a complete static analysis solution.
    
    Returns:
        A list of all analyzer labels that can be used directly with nogo.
    """
    all_analyzers = standard_go_analyzers_labels()  # type: list
    all_analyzers.extend(extended_analyzers_labels())
    return all_analyzers
