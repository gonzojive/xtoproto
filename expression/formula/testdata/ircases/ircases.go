package ircases

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/xtoproto/expression"
	"github.com/google/xtoproto/expression/formula"
	"github.com/google/xtoproto/proto/expression/formulair"
	"golang.org/x/sync/errgroup"
)

type TestCase struct {
	Name          string
	Content       string
	ASTExpression *formulair.AST_Expression
	Error         string
}

// ProtoTextName returns the name to use for the output .prototext file.
func (tc *TestCase) ProtoTextName() string {
	return tc.Name + ".prototext"
}

func (tc *TestCase) GoldenProto() *formulair.AST_TestCase {
	return &formulair.AST_TestCase{
		Name:       tc.Name,
		Error:      tc.Error,
		Expression: tc.ASTExpression,
	}
}

// Regenerate returns all of the test cases.
func Regenerate() ([]*TestCase, error) {
	entries, err := bazel.ListRunfiles()
	if err != nil {
		return nil, err
	}
	var testCases []*TestCase
	eg := &errgroup.Group{}
	for _, e := range entries {
		e := e
		if !(strings.HasPrefix(e.ShortPath, "expression/formula/testdata/ircases") && strings.HasSuffix(e.ShortPath, ".formula")) {
			continue
		}
		tc := &TestCase{
			Name: strings.TrimSuffix(path.Base(e.Path), ".formula"),
		}
		testCases = append(testCases, tc)
		eg.Go(func() error {
			contents, err := ioutil.ReadFile(e.Path)
			if err != nil {
				return fmt.Errorf("error reading file %w", err)
			}
			tc.Content = string(contents)
			regenCase(tc)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return testCases, nil
}

func regenCase(output *TestCase) {
	exp, err := expression.ParseSExpression(output.Content)
	if err != nil {
		output.Error = fmt.Sprintf("form parse error:\n%v", err)
		return
	}
	cexp, err := formula.Compile(exp)
	if err != nil {
		output.Error = fmt.Sprintf("compilation error:\n%v", err)
		return
	}
	output.ASTExpression = cexp.ASTProto()
}
