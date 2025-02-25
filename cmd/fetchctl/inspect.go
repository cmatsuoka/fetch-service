// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright 2024 Canonical Ltd.
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

package fetchctl

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"

	"github.com/canonical/fetch-service/inspectors"
	"github.com/canonical/fetch-service/inspectors/files"
	"github.com/canonical/fetch-service/metadata"
	"github.com/canonical/fetch-service/metadata/digests"
	"github.com/canonical/fetch-service/metadata/opinions"
	"github.com/canonical/fetch-service/service/config"
)

var inspectCmd InspectCmd

func init() {
	_, err := parser.AddCommand("inspect", "Run artifact inspection", "", &inspectCmd)
	if err != nil {
		panic(err)
	}
}

type InspectCmd struct {
}

type LocalArtifactInspection struct {
	ResponseInspection metadata.InspectionMap `json:"response-inspection"`
	Metadata           metadata.Metadata      `json:"metadata"`
}

func (cmd *InspectCmd) Execute(args []string) error {
	filename := args[0]
	f, err := files.OpenArtifactFile(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	digests := NewFileDigests()
	size, err := io.Copy(digests, f)
	if err != nil {
		return err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	cfg := config.InspectorsConfig{}
	insps := inspectors.New(true, cfg)

	a := metadata.NewArtifact()
	a.Metadata.Size = size
	a.Metadata.Sha1 = digests.Sha1()
	a.Metadata.Sha256 = digests.Sha256()

	// Run artifact inspectors
	if err := insps.InspectArtifact(f, a); err != nil {
		return err
	}

	// Set inspection result
	if a.Rejected() {
		a.Result = opinions.Rejected
	} else {
		a.Result = opinions.Approved
	}

	res := LocalArtifactInspection{
		ResponseInspection: a.ResponseInspection,
		Metadata:           a.Metadata,
	}

	j, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return err
	}

	fmt.Printf("%s", j)

	return nil
}

type FileDigests struct {
	sha1   hash.Hash
	sha256 hash.Hash
}

func NewFileDigests() *FileDigests {
	return &FileDigests{sha1.New(), sha256.New()}
}

func (d *FileDigests) Write(b []byte) (int, error) {
	n := len(b)
	d.sha1.Write(b[:n])
	d.sha256.Write(b[:n])
	return n, nil
}

func (d *FileDigests) Sha1() digests.Sha1Digest {
	return *(*digests.Sha1Digest)(d.sha1.Sum(nil))
}

func (d *FileDigests) Sha256() digests.Sha256Digest {
	return *(*digests.Sha256Digest)(d.sha256.Sum(nil))
}
