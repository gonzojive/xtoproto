// Program make_release helps make a release of xtoproto and can also run a
// webserver
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
)

var (
	projectDir          = flag.String("workspace", "", "path to workspace directory")
	stagingDir          = flag.String("staging", "/tmp/xtoproto-staging", "path to staging directory")
	releaseBranchSuffix = flag.String("branch_suffix", "", "suffix for git branches created during the release process")
	tag                 = flag.String("tag", "", "should be something like v0.0.5")
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	root := *projectDir
	if root == "" {
		r, err := os.Getwd()
		if err != nil {
			return nil
		}
		root = r
	}
	if err := os.MkdirAll(*stagingDir, 0770); err != nil {
		return err
	}

	glog.Infof("running commands from %s", root)
	if err := os.Chdir(root); err != nil {
		return err
	}
	got, err := exec.Command("bazel", "run", "//cmd/xtoproto_web", "--", "--output_dir", *stagingDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error generating gh-pages content: %w/%s", err, string(got))
	}
	ghPagesBranch := fmt.Sprintf("gh-pages-release%s", *releaseBranchSuffix)
	if err := run(exec.Command("git", "checkout", "--orphan", ghPagesBranch)); err != nil {
		return fmt.Errorf("failed to to create gh pages branch: %w", err)
	}
	if err := run(exec.Command("cp", "-R", filepath.Join(*stagingDir, "*"), root)); err != nil {
		return fmt.Errorf("failed to copy files to git directory: %w", err)
	}
	return nil
}

func run(c *exec.Command) error {
	return c.Run()
}
