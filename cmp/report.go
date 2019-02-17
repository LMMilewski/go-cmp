// Copyright 2017, The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package cmp

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp/internal/value"
)

type reportFlags uint64

const (
	_ reportFlags = (1 << iota) / 2

	// reportEqual reports whether the node is equal.
	// It may not be issued with reportIgnore or reportUnequal.
	reportEqual
	// reportUnequal reports whether the node is not equal.
	// It may not be issued with reportIgnore or reportEqual.
	reportUnequal
	// reportIgnore reports whether the node was ignored.
	// It may not be issued with reportEqual or reportUnequal.
	reportIgnore
)

type reporter interface {
	Option

	// PushStep is called when a tree-traversal operation is performed
	// and provides the sub-values of x and y after applying the operation.
	// The PathStep is valid until the step is popped, while the reflect.Values
	// are valid while the entire tree is still being traversed.
	//
	// Equal and Diff always call PushStep at the start to provide an
	// operation-less PathStep used to report the root values.
	PushStep(ps PathStep, x, y reflect.Value)

	// Report is called at exactly once on leaf nodes to report whether the
	// comparison identified the node as equal, unequal, or ignored.
	// A leaf node is one that is immediately preceded by and followed by
	// a pair of PushStep and PopStep calls.
	Report(reportFlags)

	// PopStep ascends back up the value tree.
	// There is always a matching pop call for every push call.
	PopStep()
}

type defaultReporter struct {
	Option

	curPath Path
	curVals [][2]reflect.Value

	diffs  []string // List of differences, possibly truncated
	ndiffs int      // Total number of differences
	nbytes int      // Number of bytes in diffs
	nlines int      // Number of lines in diffs
}

var _ reporter = (*defaultReporter)(nil)

func (r *defaultReporter) PushStep(ps PathStep, x, y reflect.Value) {
	r.curPath.push(ps)
	r.curVals = append(r.curVals, [2]reflect.Value{x, y})
}
func (r *defaultReporter) Report(f reportFlags) {
	if f == reportUnequal {
		vs := r.curVals[len(r.curVals)-1]
		r.report(vs[0], vs[1], r.curPath)
	}
}
func (r *defaultReporter) PopStep() {
	r.curPath.pop()
	r.curVals = r.curVals[:len(r.curVals)-1]
}

func (r *defaultReporter) report(x, y reflect.Value, p Path) {
	const maxBytes = 4096
	const maxLines = 256
	r.ndiffs++
	if r.nbytes < maxBytes && r.nlines < maxLines {
		sx := value.Format(x, value.FormatConfig{UseStringer: true})
		sy := value.Format(y, value.FormatConfig{UseStringer: true})
		if sx == sy {
			// Unhelpful output, so use more exact formatting.
			sx = value.Format(x, value.FormatConfig{PrintPrimitiveType: true})
			sy = value.Format(y, value.FormatConfig{PrintPrimitiveType: true})
		}
		s := fmt.Sprintf("%#v:\n\t-: %s\n\t+: %s\n", p, sx, sy)
		r.diffs = append(r.diffs, s)
		r.nbytes += len(s)
		r.nlines += strings.Count(s, "\n")
	}
}

func (r *defaultReporter) String() string {
	s := strings.Join(r.diffs, "")
	if r.ndiffs == len(r.diffs) {
		return s
	}
	return fmt.Sprintf("%s... %d more differences ...", s, r.ndiffs-len(r.diffs))
}
