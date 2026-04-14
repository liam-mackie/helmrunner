# Changelog

## 1.0.0 (2026-04-14)


### Features

* add config loading and validation for YAML definitions ([e8c55bb](https://github.com/liam-mackie/helmrunner/commit/e8c55bb6947bebbdee4b4e35c61290d1bf10a7a1))
* add Helm SDK wrapper for install and template operations ([ebf8d26](https://github.com/liam-mackie/helmrunner/commit/ebf8d2686fa2061796a8d37235c401a917c19896))
* add TUI with selection, variable input, review, and execution screens ([34abbba](https://github.com/liam-mackie/helmrunner/commit/34abbbae8faa4d0fb4062d0373e4643b11681626))
* add variable resolution and templating for definitions ([7da53e9](https://github.com/liam-mackie/helmrunner/commit/7da53e9b390bc49af6a28df4877d98e134776677))
* scaffold project with Go module and main entry point ([1c22bb5](https://github.com/liam-mackie/helmrunner/commit/1c22bb53327975356b6381c09c83323edbfdf544))
* wire up CLI entry point with TUI and template mode ([703490d](https://github.com/liam-mackie/helmrunner/commit/703490da61780a191061d364ba08269a1756fbc0))


### Bug Fixes

* resolve staticcheck lint warnings ([8ced649](https://github.com/liam-mackie/helmrunner/commit/8ced649026c8ac1d195f8c274652124207d6907e))
* update golangci-lint to v2 for Go 1.25 compatibility ([75a19ef](https://github.com/liam-mackie/helmrunner/commit/75a19efdc822f67023e1c30f0cb0966a27780723))
