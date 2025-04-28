package main

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_IntegrationTest(t *testing.T) {
	type want struct {
		file  string
		error bool
	}
	tests := []struct {
		name    string
		testDir string
		want    want
	}{
		{
			name:    "fragment test",
			testDir: "testdata/integration/fragment/",
			want: want{
				file:  "./want/query_gen.go",
				error: false,
			},
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
			if err := run(); (err != nil) != tt.want.error {
				t.Errorf("run() error = %v, wantErr %v", err, tt.want.error)
			}

			// 生成されたファイルとwantファイルの内容を比較
			actualFilePath := "domain/query_gen.go"
			wantFilePath := tt.want.file

			// 両方のファイルを読み込む
			actualContent, err := os.ReadFile(actualFilePath)
			if err != nil {
				t.Errorf("ファイル読み込みエラー（実際のファイル）: %v", err)
				return
			}

			wantContent, err := os.ReadFile(wantFilePath)
			if err != nil {
				t.Errorf("ファイル読み込みエラー（期待されるファイル）: %v", err)
				return
			}

			// ファイルの内容を比較
			if diff := cmp.Diff(string(wantContent), string(actualContent)); diff != "" {
				t.Errorf("ファイルの内容が異なります:\n%s", diff)
			}
		})
	}
}
