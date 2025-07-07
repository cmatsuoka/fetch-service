// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright 2025 Canonical Ltd.
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

package snap_test

import (
	"github.com/gabriel-vasile/mimetype"
	. "gopkg.in/check.v1"

	"github.com/canonical/fetch-service/inspectors/files"
	"github.com/canonical/fetch-service/inspectors/snap"
	"github.com/canonical/fetch-service/metadata"
	"github.com/canonical/fetch-service/metadata/opinions"
)

func (s *snapSuite) TestSnapNonceInspectorID(c *C) {
	ins := snap.NewSnapNonceInspector()
	c.Assert(ins.ID(), Equals, "snap.nonce")
}

type snapNonceInspectRequestTest struct {
	url     string // The request URL
	pending bool   // The expected inspection result
	reason  string // The reason for the inspection result
}

var snapNonceInspectRequestTests = []snapNonceInspectRequestTest{{
	url:     "https://api.snapcraft.io:443/v1/snaps/auth/nonces",
	pending: true,
	reason:  "valid URL for snap nonce",
}, {
	url:     "https://api.snapcraft.io:443/v2/something-else", // different path
	pending: false,
}, {
	url:     "https://api.snapcraft.io:443/v2/snaps/auth/nonces", // different version
	pending: false,
}, {
	url:     "http://api.snapcraft.io/v1/snaps/auths/nonces", // different protocol
	pending: false,
}}

func (s *snapSuite) TestSnapNonceInspectRequest(c *C) {
	for _, tc := range snapNonceInspectRequestTests {
		ins := snap.NewSnapNonceInspector()
		a := metadata.NewArtifact()
		a.CurrentDownload = metadata.Download{URL: tc.url}

		err := ins.InspectRequest(a)
		c.Assert(err, IsNil)

		insp, ok := a.RequestInspection[ins.ID()]
		c.Assert(ok, Equals, tc.pending, Commentf("test case: %+v", tc))
		if ok {
			c.Assert(insp.Opinion, Equals, opinions.Pending)
			c.Assert(insp.Reason, Equals, tc.reason)
		}
	}
}

type snapNonceArtifactInspectorTest struct {
	filename string // The path to the artifact to be tested
	approved bool   // Whether this artifact is expected to be approved
	reason   string // The reason for approval or rejection
	filetype string // The expected file type
}

var snapNonceArtifactInspectorTests = []snapNonceArtifactInspectorTest{{
	filename: "testdata/nonce.json",
	approved: true,
	reason:   "valid snap nonce",
	filetype: "application/x.canonical.snap-nonce",
}, {
	filename: "testdata/nonce-bad-1.json",
	approved: false,
}, {
	filename: "testdata/nonce-bad-2.json",
	approved: false,
}, {
	filename: "testdata/nonce-bad-3.json",
	approved: false,
}, {
	filename: "testdata/snap-revision.assert",
	approved: false,
}, {
	filename: "testdata/snap-declaration.assert",
	approved: false,
}, {
	filename: "testdata/account.assert",
	approved: false,
}, {
	filename: "testdata/account-key.assert",
	approved: false,
}}

func (s *snapSuite) TestSnapNonceArtifactInspector(c *C) {
	for _, tc := range snapNonceArtifactInspectorTests {
		a := metadata.NewArtifact()
		a.Metadata.Type = "application/json"
		a.Metadata.Size = 3330
		a.MimeType = mimetype.Lookup("application/json")

		f, err := files.OpenArtifactFile(tc.filename)
		c.Assert(err, IsNil)
		defer f.Close()

		ins := snap.NewSnapNonceInspector()
		a.SetRequestPending(ins, "test")
		err = ins.InspectArtifact(f, a)
		c.Assert(err, IsNil)
		c.Assert(a.Approved(), Equals, tc.approved, Commentf("test case: %+v", tc))

		if tc.approved {
			c.Check(a.ResponseInspection["snap.nonce"].Reason, Equals, tc.reason)
			c.Check(a.Metadata.Type, Equals, tc.filetype)
			c.Check(a.Metadata.Name, Equals, "nonce")
			c.Check(a.Metadata.Description, Equals, "Snap nonce JSON response")
			c.Check(a.Metadata.Size, Equals, int64(3330))
		}
	}
}
