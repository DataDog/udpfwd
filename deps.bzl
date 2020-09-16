load("@bazel_gazelle//:deps.bzl", "go_repository")

def deps():
    go_repository(
        name = "com_github_datadog_datadog_go",
        importpath = "github.com/DataDog/datadog-go",
        sum = "h1:Dq8Dr+4sV1gBO1sHDWdW+4G+PdsA+YSJOK925MxrrCY=",
        version = "v4.0.0+incompatible",
    )
