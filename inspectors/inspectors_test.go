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

package inspectors_test

import (
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"

	. "github.com/canonical/fetch-service/inspectors/common"
	"github.com/canonical/fetch-service/logger"
	"github.com/canonical/fetch-service/logger/testlogger"
	"github.com/canonical/fetch-service/metadata"
	"github.com/canonical/fetch-service/metadata/digests"
	"github.com/canonical/fetch-service/metadata/opinions"
	"github.com/canonical/fetch-service/session"
)

const (
	MySha256 = "c1de7d7ad587318b4674ed029c7d22e33ce90268ca32c5b3dd1cff36511c7950"
)

func Test(t *testing.T) { TestingT(t) }

type inspectorsSuite struct{}

func (t *inspectorsSuite) SetUpTest(c *C) {
	testlogger.Init(logger.InfoLevel)
}

var _ = Suite(&inspectorsSuite{})

func (t *inspectorsSuite) TestRunRequestInspectors(c *C) {
	for _, tc := range []struct {
		url    string
		errMsg string
	}{
		{"http://some.url", ""},
		{":not-a-url", "cannot parse URL:.*"},
	} {
		a := metadata.NewArtefact()
		a.CurrentDownload.URL = tc.url

		s := session.New(c.MkDir(), false)
		defer s.Discard()

		err := s.Insps.RunRequestInspectors(a)

		if tc.errMsg == "" {
			c.Assert(err, IsNil)
			c.Assert(len(a.RequestInspection), Equals, 1)
			c.Assert(a.RequestInspection["default"], DeepEquals, &Inspection{
				Opinion: opinions.Unknown,
				Reason:  "the request was not recognized by any format inspector",
			})
		} else {
			c.Assert(err, ErrorMatches, tc.errMsg)
		}
	}
}

func (t *inspectorsSuite) TestRunArtefactInspectors(c *C) {
	for _, tc := range []struct {
		permissive bool
		pending    bool // Whether the request opinion is pending
		fileExists bool
		errMsg     string
	}{
		{true, true, true, ""},
		{true, true, false, "open .*: no such file or directory"},
		{true, false, true, ""},
		{false, true, true, ""},
		{false, false, true, ""}, // failed request inspection in strict mode
	} {
		dir := c.MkDir()
		data := []byte("Measure twice, saw once.\n")
		if tc.fileExists {
			err := os.WriteFile(filepath.Join(dir, "c1de7d7ad587318b4674ed029c7d22e33ce90268ca32c5b3dd1cff36511c7950.data"), data, 0644)
			c.Assert(err, IsNil)
		}

		h, _ := digests.NewSha256Digest(MySha256)
		a := metadata.NewArtefact()
		a.CurrentDownload.ContentType = "text/plain"
		a.CurrentDownload.Sha256 = h
		a.CurrentDownload.URL = "http://some.url"
		a.Metadata.Sha256 = h

		s := session.New(dir, tc.permissive)
		defer s.Discard()

		err := s.Insps.RunArtefactInspectors(dir, a)

		if tc.errMsg == "" {
			c.Assert(err, IsNil)
			c.Check(a.Metadata.Type, Equals, "text/plain; charset=utf-8")
			c.Check(len(a.ResponseInspection), Equals, 1)
			c.Check(a.ResponseInspection["default"], DeepEquals, &Inspection{
				Opinion: opinions.Unknown,
				Reason:  "the artefact format is unknown",
			})
		} else {
			c.Assert(err, ErrorMatches, tc.errMsg)
		}
		c.Assert(a.Rejected(), Equals, true)
	}
}

func (t *inspectorsSuite) TestList(c *C) {
	s := session.New("", true)
	defer s.Discard()

	insps := s.Insps.List()
	c.Assert(len(insps), Equals, 18)
}
