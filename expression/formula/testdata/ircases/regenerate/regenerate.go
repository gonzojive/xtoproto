package main

import (
	"flag"
	"io/ioutil"
	"path"

	"github.com/golang/glog"
	"github.com/google/xtoproto/expression/formula/testdata/ircases"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	outputDir = flag.String("output_dir", "", "output directory")
)

var marshalOptions = prototext.MarshalOptions{
	Multiline: true,
	Indent:    "  ",
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		glog.Errorf("%v", err)
	}
}

func run() error {
	cases, err := ircases.Load(ircases.LoadOptions{Regenerate: true})
	if err != nil {
		return err
	}
	eg := &errgroup.Group{}
	for _, tc := range cases {
		tc := tc
		eg.Go(func() error {
			outputPath := path.Join(*outputDir, tc.ProtoTextName())
			goldenContent := marshalOptions.Format(tc.RegeneratedProto())
			return ioutil.WriteFile(outputPath, []byte(goldenContent), 0664)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	glog.Infof("wrote %d files", len(cases))
	return nil
}
