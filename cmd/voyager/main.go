// Copyright 2020 The Infinity Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/yanhuangpai/voyager/cmd/voyager/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		println(os.Stderr, "Error:", err.Error())
		os.Exit(1)
	}
}
