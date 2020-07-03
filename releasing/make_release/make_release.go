// Program make_release helps make a release of xtoproto and can also run a
// webserver
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/golang/glog"
)

var (
	projectDir = flag.String("workspace", "", "path to workspace directory")
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

	glog.Infof("running commands from %s", root)
	if err := os.Chdir(root); err != nil {
		return err
	}
	got, err := exec.Command("bazel", "run", "//cmd/xtoproto_web", "--", "--output_dir", root).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error generating gh-pages content: %w/%s", err, string(got))
	}
	return nil
}
