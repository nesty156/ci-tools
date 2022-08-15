package main

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGatherOptions(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected options
	}{
		{
			name: "default",
			args: []string{"cmd"},
			expected: options{
				runOnce:           false,
				dryRun:            true,
				intervalRaw:       "1h",
				cacheFile:         "",
				cacheRecordAgeRaw: "168h",
				configFile:        "",
			},
		},
		{
			name: "basic case",
			args: []string{"cmd", "--run-once=true", "--interval=2h", "--cache-file=cache.yaml", "--cache-record-age=100h", "--config-file=config.yaml"},
			expected: options{
				runOnce:           true,
				dryRun:            true,
				intervalRaw:       "2h",
				cacheFile:         "cache.yaml",
				cacheRecordAgeRaw: "100h",
				configFile:        "config.yaml",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = tc.args
			actual := gatherOptions()

			if diff := cmp.Diff(tc.expected.runOnce, actual.runOnce); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
			if diff := cmp.Diff(tc.expected.dryRun, actual.dryRun); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
			if diff := cmp.Diff(tc.expected.intervalRaw, actual.intervalRaw); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
			if diff := cmp.Diff(tc.expected.cacheFile, actual.cacheFile); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
			if diff := cmp.Diff(tc.expected.cacheRecordAgeRaw, actual.cacheRecordAgeRaw); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
			if diff := cmp.Diff(tc.expected.configFile, actual.configFile); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
		})
	}
}

/* func TestValidate(t *testing.T) {
	os.Args = []string{"cmd", "--cache-record-age=100h", "--interval=2h", "--config-file=config.yaml"}
	testCases := []struct {
		name     string
		o        options
		expected error
	}{
		{
			name: "config",
			o: options{
				runOnce:           false,
				dryRun:            true,
				intervalRaw:       "1h",
				cacheFile:         "",
				cacheRecordAgeRaw: "168h",
				configFile:        "",
			},
			expected: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.o.Validate()
			if diff := cmp.Diff(tc.expected, err, testhelper.EquateErrorMessage); diff != "" {
				t.Errorf("Error differs from expected:\n%s", diff)
			}
		})
	}
} */
