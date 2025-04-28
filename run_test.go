package main

import (
	"os"
	"testing"
)

func Test_IntegrationTest(t *testing.T) {
	tests := []struct {
		name    string
		testDir string
		wantErr bool
	}{
		{
			name:    "fragment test",
			testDir: "testdata/integration/fragment/",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic: %v", r)
				}
			}()

			if err := os.Chdir(tt.testDir); err != nil {
				t.Errorf("run() error = %v", err)
			}
			if err := run(); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
