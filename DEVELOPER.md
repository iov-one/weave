# Useful tips

## How to make protobuf imports work in your IDE
1. `go list -f '{{ .Dir }}' -m github.com/gogo/protobuf`
2. Add the directory from step 1 to your protobuf imports in the plugin you use
3. Repeat 1 and 2 if the dependency version has changed and things no longer work
