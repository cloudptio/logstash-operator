// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package name

import (
	"testing"
)

func TestHTTPService(t *testing.T) {
	type args struct {
		lsName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "sample",
			args: args{lsName: "sample"},
			want: "sample-ls-http",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPService(tt.args.lsName); got != tt.want {
				t.Errorf("HTTPService() = %v, want %v", got, tt.want)
			}
		})
	}
}
