"""Nogo dependency labels for Go analyzers."""

def generate_analyzers_labels(base_path, items):
    """Generates labels by prefixing each item with a base path.

    Args:
        base_path: The common prefix for all labels.
        items: A list of items to be converted into labels.

    Returns:
        A list of generated labels.
    """

    labels = []
    for item in items:
        label = base_path + item
        labels.append(label)

    return labels

def go_vet_analyzers_labels():
    """Generates nogo dependency labels for Go vet analyzers.

    Source: 
        Vet Analyzers: https://cs.opensource.google/go/go/+/refs/tags/go1.22.5:src/cmd/vet/main.go;l=12-43

    Returns:
        labels: A list of dependency labels for Go vet analyzers.
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
