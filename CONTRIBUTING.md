# Contributing

Pull requests and Github issues are welcome!
Ask **questions** by posting to the [H3 tag on StackOverflow](https://stackoverflow.com/questions/tagged/h3)

## Pull requests

* Please include tests that show the bug is fixed or feature works as intended.
* Please add a description of your change to the Unreleased section of the
  [changelog](./CHANGELOG.md).
* Please open issues to discuss large features or changes which would break
  compatibility, before submitting pull requests.
* Please keep code coverage of H3-Go library near 100%.

## Development

Install dependencies:

```bash
go get -t ./...
```

Run tests:

```bash
go test
```

Generate coverage:

```bash
go test -coverprofile=covprofile && go tool cover -html=covprofile
```

### Using VSCode

Add VSCode configuration, or merge recommended settings into your existing settings:

```sh
git cherry-pick vscode
```

`golangci-lint` version 2 requires the pre-release version of the `golang.go` extension:

https://golangci-lint.run/welcome/integrations/#visual-studio-code

## Other ways to contribute

You may also be interested in [contributing to the @uber/h3
repository](https://github.com/uber/h3/blob/master/CONTRIBUTING.md), which
includes more detailed documentation for the functions provided by H3-Go.

