# contributing

Contribute to this library by submitting PR (pull requests).

Make sure your contribution passes the following validations:

1. New code must be formatted according to `fmt` standards:

   `go fmt ./...`

2. New code must pass `lint` validations:

   `golint ./...`

3. New code must pass Go Vetting practices:

   `go vet ./...`

4. And new code must pass [staticcheck](https://godoc.org/honnef.co/go/tools/cmd/staticcheck) checks:

   `staticcheck ./...`

I would like to keep this library simple, the proposed change must be a common use case.

## maintainer

Code is organized in a flat module space, the public API of the core module is intentionally small, with public
structures that expose only one public method always with the signature `(*struct) Apply() error`.

### example

```
type EnforceLaunchConfig struct {
    // ...
}
    EnforceLaunchConfig encapsulates the attributes of a LaunchConfig
    enforcement

func (e *EnforceLaunchConfig) Apply() error
    Apply the LaunchConfig enforcement
```

Packages `cmd/*` are entry points, `main` packages which make use of the public structures in the core module. Main
packages are purposely simple and only make use of one core structure, their job is just create a new structure run the
`Apply` method and handle the returned `error`.

## releases

Create a GitHub release, run `./make.sh` and attach the deliverables (zip files) to the release.

