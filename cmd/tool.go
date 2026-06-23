// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lindb/common/pkg/fileutil"
)

// errSkipped is returned by checkExistenceOf when the user declines to overwrite.
// Callers should treat this as a clean exit, not an error condition.
var errSkipped = errors.New("skipped")

// askForConfirmation reads a y/n answer from stdin, looping until a valid answer is given.
func askForConfirmation() (bool, error) {
	for {
		_, _ = fmt.Fprintf(os.Stdout, "[y/n]: ")
		var answer string
		if _, err := fmt.Scanln(&answer); err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please answer y or n.")
		}
	}
}

// checkExistenceOf prompts the user for confirmation before overwriting an existing file.
// Returns errSkipped when the user declines; callers should exit cleanly in that case.
func checkExistenceOf(path string) error {
	if fileutil.Exist(path) {
		fmt.Println("An old config already exists, do you want to overwrite it?")
		confirmed, err := askForConfirmation()
		if err != nil {
			return err
		}
		if confirmed {
			fmt.Println("Overwriting...")
		} else {
			fmt.Println("Skipping...")
			return errSkipped
		}
	}
	return nil
}
