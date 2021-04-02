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
	"google.golang.org/protobuf/encoding/prototext"
)

const (
	inputSuffix  = ".formula"
	goldenSuffix = ".golden.pbtxt"
)

type TestCase struct {
	goldenProto *formulair.AST_TestCase
	regenerated *formulair.AST_TestCase

	inputPath string
}

// Name returns the name of the test case.
func (tc *TestCase) Name() string {
	return strings.TrimSuffix(path.Base(tc.inputPath), inputSuffix)
}

// InputFormula returns the input to the test case.
func (tc *TestCase) InputFormula() string {
	return tc.regenerated.GetInputFormula()
}

// ProtoTextName returns the name to use for the output .prototext file.
func (tc *TestCase) ProtoTextName() string {
	return tc.Name() + goldenSuffix
}

// RegeneratedProto returns the regenerated test case proto.
func (tc *TestCase) RegeneratedProto() *formulair.AST_TestCase {
	return tc.regenerated
}

// GoldenProto returns the already generated test case proto.
func (tc *TestCase) GoldenProto() *formulair.AST_TestCase {
	return tc.goldenProto
}

// LoadOptions parameterizes Load.
type LoadOptions struct {
	Regenerate, LoadGoldens bool
}

// Load returns all of the test cases.
func Load(opts LoadOptions) ([]*TestCase, error) {
	cases, err := loadInputs()
	if err != nil {
		return nil, err
	}
	eg := &errgroup.Group{}
	for _, tc := range cases {
		tc := tc
		if opts.LoadGoldens {
			eg.Go(func() error {
				return loadGolden(tc)
			})
		}
		if opts.Regenerate {
			eg.Go(func() error {
				regenCase(tc)
				return nil
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return cases, nil
}

func loadInputs() ([]*TestCase, error) {
	entries, err := bazel.ListRunfiles()
	if err != nil {
		return nil, err
	}
	var testCases []*TestCase
	eg := &errgroup.Group{}
	for _, e := range entries {
		e := e
		if !(strings.HasPrefix(e.ShortPath, "expression/formula/testdata/ircases") && strings.HasSuffix(e.ShortPath, inputSuffix)) {
			continue
		}
		tc := &TestCase{
			inputPath: e.Path,
		}
		testCases = append(testCases, tc)
		eg.Go(func() error {
			contents, err := ioutil.ReadFile(tc.inputPath)
			if err != nil {
				return fmt.Errorf("error reading file %w", err)
			}
			tc.regenerated = &formulair.AST_TestCase{
				Name:         tc.Name(),
				InputFormula: string(contents),
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return testCases, nil
}

func loadGolden(tc *TestCase) error {
	goldenPrototext, err := ioutil.ReadFile(strings.TrimSuffix(tc.inputPath, inputSuffix) + goldenSuffix)
	if err != nil {
		return fmt.Errorf("error reading file %w", err)
	}
	golden := &formulair.AST_TestCase{}
	if err := prototext.Unmarshal(goldenPrototext, golden); err != nil {
		return fmt.Errorf("error loading case %s: %w", tc.Name, err)
	}
	tc.goldenProto = golden
	return nil
}

func regenCase(tc *TestCase) {
	exp, err := expression.ParseSExpression(tc.InputFormula())
	if err != nil {
		tc.regenerated.Error = fmt.Sprintf("form parse error:\n%v", err)
		return
	}
	cexp, err := formula.Compile(exp)
	if err != nil {
		tc.regenerated.Error = fmt.Sprintf("compilation error:\n%v", err)
		return
	}
	tc.regenerated.Expression = cexp.ASTProto()
}
