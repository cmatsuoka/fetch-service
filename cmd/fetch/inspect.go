// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright 2023-2024 Canonical Ltd.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/canonical/fetch-service/inspectors"
	"github.com/canonical/fetch-service/metadata"
)

func inspectArtefact(filename string) error {
	dir, err := os.MkdirTemp("", "inspect-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	fin, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.CreateTemp(dir, "temp-")
	if err != nil {
		return err
	}

	// Copy file and compute digest
	h := sha256.New()
	r := io.TeeReader(fin, h)
	n, err := io.Copy(fout, r)
	if err != nil {
		return err
	}
	fout.Close()

	a := metadata.NewArtefact()
	a.Metadata.Size = n
	copy(a.Metadata.Sha256[:], h.Sum(nil))

	fmt.Printf("renaming %s to %s\n", fout.Name(), filepath.Join(dir, a.Metadata.Sha256.String()+".data"))
	if err := os.Rename(fout.Name(), filepath.Join(dir, a.Metadata.Sha256.String()+".data")); err != nil {
		return err
	}

	insps := inspectors.New(true)
	if err := insps.RunArtefactInspectors(dir, a); err != nil {
		return err
	}

	j, err := json.MarshalIndent(a, "", "\t")
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", j)

	return nil
}
