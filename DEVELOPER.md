# Useful tips

## How to make protobuf imports work in your IDE
1. `go list -f '{{ .Dir }}' -m github.com/gogo/protobuf`
2. Add the directory from step 1 to your protobuf imports in the plugin you use
3. Repeat 1 and 2 if the dependency version has changed and things no longer work

## Specs

The `spec` directory holds auto-generated files intended to be used by other
projects that import weave. The follow subdirectories have the following
information:

* `gogo` - The original protobuf definitions (no code), including all gogo/protobuf-specific directives
* `proto` - The cleaned protobuf definitions, binary compatible with `gogo`, but able to be compiled for any language