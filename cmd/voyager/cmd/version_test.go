// Copyright 2020 The Infinity Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd_test

import (
	"bytes"
	"testing"

	"github.com/yanhuangpai/voyager"
	"github.com/yanhuangpai/voyager/cmd/voyager/cmd"
)

func TestVersionCmd(t *testing.T) {
	var outputBuf bytes.Buffer
	if err := newCommand(t,
		cmd.WithArgs("version"),
		cmd.WithOutput(&outputBuf),
	).Execute(); err != nil {
		t.Fatal(err)
	}

	want := voyager.Version + "\n"
	got := outputBuf.String()
	if got != want {
		t.Errorf("got output %q, want %q", got, want)
	}
}
