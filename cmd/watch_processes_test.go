/*
Copyright © 2024 Aaron Lifton <aaronlifton@gmail.com>
*/
package cmd

import (
	"reflect"
	"testing"
	// "github.com/mitchellh/go-ps"
	// "github.com/spf13/cobra"
)

func Test_NewWatchProcessesCmd(t *testing.T) {
	tests := []struct {
		name string
		pids []int
	}{
		{
			name: "returns the right pids",
			pids: []int{1, 2, 3},
		},
	}
	cmd := RunWatchProcesses()
	got := cmd
	for _, tt := range tests {
		want := tt.pids
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", tt.name, got, want)
		}
	}
}
