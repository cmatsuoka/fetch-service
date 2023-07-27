// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright 2023 Canonical Ltd.
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

package metadata_test

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "gopkg.in/check.v1"

	"github.com/canonical/fetch-service/metadata"
)

type aptSuite struct{}

var _ = Suite(&aptSuite{})

const (
	releaseURL  = "http://archive.ubuntu.com/ubuntu/dists/jammy/InRelease"
	packagesURL = "http://archive.ubuntu.com/ubuntu/dists/jammy/main/binary-amd64/Packages.xz"
)

// XXX: This file contains minimal testing for apt file formats. Tests
//      will be extended after the metadata format is approved.

func (s *aptSuite) TestAptReleaseInspector(c *C) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	req, err := http.NewRequest("GET", releaseURL, nil)
	c.Assert(err, IsNil)

	resp, err := client.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, 200)

	defer resp.Body.Close()

	tmp := c.MkDir()
	dest, err := os.Create(filepath.Join(tmp, "290d07339dde2735121ab03e525ca6593c395a42.bin"))
	c.Assert(err, IsNil)

	_, err = io.Copy(dest, resp.Body)
	c.Assert(err, IsNil)

	dest.Close()

	h, _ := metadata.NewSha1Digest("290d07339dde2735121ab03e525ca6593c395a42")
	md := &metadata.Metadata{Type: "application/x-apt-release", Sha1: h}
	di := &metadata.DownloadInfo{}

	var iface metadata.Inspector
	ins := metadata.AptReleaseInspector{}
	c.Assert(ins, Implements, &iface)

	ctx := metadata.NewInspectionContext()

	stop, err := ins.Inspect(filepath.Join(tmp, "290d07339dde2735121ab03e525ca6593c395a42.bin"), md, di, ctx)
	c.Assert(err, IsNil)
	c.Assert(stop, Equals, true)

	c.Check(md.Name, Equals, "InRelease")
	c.Check(md.Vendor, Equals, "Ubuntu")
	c.Check(md.Description, Equals, "Ubuntu Jammy 22.04")
	c.Check(md.Author, Equals, "Ubuntu")
	c.Check(md.Annotations, HasLen, 0)
}

func (s *aptSuite) TestAptPackagesInspector(c *C) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	req, err := http.NewRequest("GET", packagesURL, nil)
	c.Assert(err, IsNil)

	resp, err := client.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, 200)

	defer resp.Body.Close()

	tmp := c.MkDir()
	dest, err := os.Create(filepath.Join(tmp, "0f9d4626df5afdf378004213b7f594cfb1ca0159ad00a4921fb40049dbcb292e.data"))
	c.Assert(err, IsNil)

	size, err := io.Copy(dest, resp.Body)
	c.Assert(err, IsNil)

	dest.Close()

	// simulate metadata collected from InRelease
	p := metadata.AptReleasePackages{
		Path:   "dists/test/Packages.xz",
		Vendor: "Acme",
		Size:   size,
	}

	releaseHash, _ := metadata.NewSha256Digest("7a0965cdce7e57af669e786379edcf45953de9bca3763342b870b3ce6d0dd777")
	packagesHash, _ := metadata.NewSha256Digest("0f9d4626df5afdf378004213b7f594cfb1ca0159ad00a4921fb40049dbcb292e")
	ctx := metadata.NewInspectionContext()
	metadata.EnsureAptContext(ctx)
	metadata.GetAptContext(ctx).AddReleasePackages(releaseHash, packagesHash, p)

	md := &metadata.Metadata{
		Type:   "application/x-apt-packages",
		Sha256: packagesHash,
		Size:   size,
	}
	di := &metadata.DownloadInfo{}

	var iface metadata.Inspector
	ins := metadata.AptPackagesInspector{}
	c.Assert(ins, Implements, &iface)

	stop, err := ins.Inspect(filepath.Join(tmp, "0f9d4626df5afdf378004213b7f594cfb1ca0159ad00a4921fb40049dbcb292e.data"), md, di, ctx)
	c.Assert(err, IsNil)
	c.Assert(stop, Equals, true)

	c.Check(md.Name, Equals, "dists/test/Packages.xz")
	c.Check(md.Vendor, Equals, "Acme")
	c.Check(md.Description, Equals, "Apt repository Packages file")
	c.Check(md.Author, Equals, "Acme")
	c.Check(md.Annotations["file.integrity.asserted-by"].Kind, Equals, metadata.Notice)
	c.Check(md.Annotations["file.integrity.asserted-by"].Value, Equals, "7a0965cdce7e57af669e786379edcf45953de9bca3763342b870b3ce6d0dd777")
}

func (s *aptSuite) TestContextReleasePackages(c *C) {
	ctx := metadata.NewInspectionContext()
	c.Assert(ctx, Not(IsNil))

	metadata.EnsureAptContext(ctx)

	p := metadata.AptReleasePackages{
		Path:   "path/to/Packages.xz",
		Size:   12345,
		Vendor: "Acme",
	}

	releaseDigest, _ := metadata.NewSha256Digest(MySha256)
	packagesDigest, _ := metadata.NewSha256Digest("f1d6e0e435c851796ddc982230070bf5f6c313fade049f31e2983e5b26c43a72")
	otherDigest, _ := metadata.NewSha256Digest("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	metadata.GetAptContext(ctx).AddReleasePackages(releaseDigest, packagesDigest, p)

	digest, _, ok := metadata.GetAptContext(ctx).GetReleasePackages(otherDigest)
	c.Assert(ok, Equals, false)
	c.Assert(digest, Equals, metadata.Sha256Digest{})

	digest, q, ok := metadata.GetAptContext(ctx).GetReleasePackages(packagesDigest)
	c.Assert(ok, Equals, true)
	c.Assert(digest, Equals, releaseDigest)
	c.Assert(q, DeepEquals, p)
}

func (s *aptSuite) TestContextPackagesEntry(c *C) {
	ctx := metadata.NewInspectionContext()
	c.Assert(ctx, Not(IsNil))

	metadata.EnsureAptContext(ctx)

	e := metadata.AptPackagesEntry{
		Package:      "hello",
		Version:      "1.2.3",
		Architecture: "amd64",
		Size:         1337,
	}

	packagesDigest, _ := metadata.NewSha256Digest(MySha256)
	helloDigest, _ := metadata.NewSha256Digest("e24f8496e591bfa9fc493ab6bbb702b8ee60a47d974139c17f20f095dd0d5670")
	otherDigest, _ := metadata.NewSha256Digest("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	metadata.GetAptContext(ctx).AddPackagesEntry(packagesDigest, helloDigest, e)

	digest, _, ok := metadata.GetAptContext(ctx).GetPackagesEntry(otherDigest)
	c.Assert(ok, Equals, false)
	c.Assert(digest, Equals, metadata.Sha256Digest{})

	digest, f, ok := metadata.GetAptContext(ctx).GetPackagesEntry(helloDigest)
	c.Assert(ok, Equals, true)
	c.Assert(digest, Equals, packagesDigest)
	c.Assert(f, DeepEquals, e)
}
