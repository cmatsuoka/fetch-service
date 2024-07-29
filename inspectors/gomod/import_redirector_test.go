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

package gomod_test

import (
	"bytes"
	"net/http"
	"os"

	"github.com/gabriel-vasile/mimetype"
	. "gopkg.in/check.v1"

	. "github.com/canonical/fetch-service/inspectors/common"
	"github.com/canonical/fetch-service/inspectors/gomod"
	"github.com/canonical/fetch-service/metadata"
)

type importRedirectorSuite struct{}

var _ = Suite(&importRedirectorSuite{})

func (s *importRedirectorSuite) TestImportRedirectorInspectorInterface(c *C) {
	var iface Inspector
	ins := gomod.NewImportRedirectorInspector()
	c.Assert(ins, Implements, &iface)

}

func (s *importRedirectorSuite) TestImportRedirectorInspectorID(c *C) {
	ins := gomod.NewImportRedirectorInspector()
	c.Assert(ins.ID(), Equals, "go.import-redirector")

}

func (s *importRedirectorSuite) TestInspectImportRedirectorRequest(c *C) {
	for _, tc := range []struct {
		url    string
		result bool
	}{
		// FIXME: using github as placeholder, final URLs will change
		{"https://github.com:443/user/project.git?go-get=1", true},
		{"https://github.com:443/user/project?go-get=1", true},
		{"https://gopkg.in:443/project.v2/?go-get=1", true},
		{"https://invalid.com:443/user/project.git?go-get=1", false},
		{"http://github.com/user/project.git?go-get=1", false},
		{"https://gothub.com:443/user/project.git?go-get=1", false},
		{"ahttps://github.com:443/user/project.git?go-get=1", false},
	} {
		ins := gomod.NewImportRedirectorInspector()
		a := metadata.NewArtefact()
		a.CurrentDownload.URL = tc.url
		a.Request, _ = http.NewRequest("GET", tc.url, nil)

		err := ins.InspectRequest(a)
		c.Assert(err, IsNil)

		c.Assert(a.RequestPending(), Equals, tc.result)
	}
}

func (s *importRedirectorSuite) TestImportRedirectorInspectArtefact(c *C) {
	for _, tc := range []struct {
		mimetype string
		metaName []byte
		approved bool
	}{
		{"text/html", []byte(`meta name="go-import"`), true},
		{"text/html", []byte(`meta name="something-else"`), false},
		{"text/plain", []byte(`meta name="go-import"`), false},
	} {
		a := metadata.NewArtefact()
		a.Request, _ = http.NewRequest("GET", "https://example.com:443/test?go-get=1", nil)
		a.CurrentDownload.ContentType = tc.mimetype
		a.MimeType = mimetype.Lookup(tc.mimetype)

		data, err := os.ReadFile("testdata/index.data")
		c.Assert(err, IsNil)
		data = bytes.Replace(data, []byte(`meta name="go-import`), tc.metaName, 1)
		f := bytes.NewReader(data)

		ins := gomod.NewImportRedirectorInspector()
		a.SetRequestPending(ins, "test")
		err = ins.InspectArtefact(f, a)
		c.Assert(err, IsNil)

		c.Assert(a.Approved(), Equals, tc.approved)
	}
}
