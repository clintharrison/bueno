//go:build tools

// https://github.com/bazel-contrib/rules_go/blob/42a97e95154170f6be04cbbdbe87808016e4470f/docs/go/core/bzlmod.md#depending-on-tools
package deps

import (
	_ "github.com/nishanths/exhaustive"
	_ "github.com/sluongng/nogo-analyzer/staticcheck"
	_ "golang.org/x/tools/go/analysis"
	_ "honnef.co/go/tools/staticcheck"
)
