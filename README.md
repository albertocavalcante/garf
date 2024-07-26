# garf

garf is a CLI to help with mirroring artifacts to JFrog Generic Repository.

## Running

```bat
set JFROG_URL=https://albertocavalcante.jfrog.io/artifactory
set JFROG_USER=user
set JFROG_PASSWORD=password
```

```sh
bazel run //:garf -- --source https://github.com/bazelbuild/bazel/releases/download/7.2.1/bazel_nojdk-7.2.1-windows-x86_64.exe --destination sandbox-generic-local
```
