package query

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/privacy-sandbox-aggregation-service/pipeline/dpfaggregator"
)

func TestExpansionConfigReadWrite(t *testing.T) {
	tmpDir, err := ioutil.TempDir("/tmp", "test-config")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &ExpansionConfig{
		PrefixLengths:               []int32{1, 2, 3},
		ExpansionThresholdPerPrefix: []uint64{4, 5},
	}
	configFile := path.Join(tmpDir, "config_file")
	if err := WriteExpansionConfigFile(config, configFile); err != nil {
		t.Fatal(err)
	}
	got, err := ReadExpansionConfigFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(config, got); diff != "" {
		t.Errorf("expansion config read/write mismatch (-want +got):\n%s", diff)
	}
}

func TestGetNextNonemptyPrefixes(t *testing.T) {
	result := []dpfaggregator.CompleteHistogram{
		{Index: 1, Sum: 2, Count: 3},
		{Index: 2, Sum: 3, Count: 4},
		{Index: 3, Sum: 4, Count: 5},
	}
	got := getNextNonemptyPrefixes(result, 3)
	want := []uint64{2, 3}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("nonempty prefixes mismatch (-want +got):\n%s", diff)
	}
}
