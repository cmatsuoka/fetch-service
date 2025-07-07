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

package snap

import (
	"encoding/json"
	"fmt"
	"net/url"

	. "github.com/canonical/fetch-service/inspectors/common"
	"github.com/canonical/fetch-service/inspectors/mimetypes"
)

// SnapNonceInspector examines assertion requests and files.
//
// Assertions are text-based and take a context-dependent format
// that always includes one or more headers, an optional body, and
// the encoded signature.
type SnapNonceInspector struct {
}

func NewSnapNonceInspector() *SnapNonceInspector {
	return &SnapNonceInspector{}
}

func (SnapNonceInspector) ID() string {
	return "snap.nonce"
}

// InspectRequest verifies if the request complies with policy.
func (ins *SnapNonceInspector) InspectRequest(a RequestArtifact) error {
	u, err := url.Parse(a.DownloadURL())
	if err != nil {
		return fmt.Errorf("cannot parse URL: %s", err)
	}

	if _, err := newSnapNonceUrlInfo(u); err == nil {
		a.SetRequestPending(ins, "valid URL for snap nonce")
	}

	return nil
}

// InspectArtifact extracts metadata from a known artifact file format.
func (ins *SnapNonceInspector) InspectArtifact(f ArtifactReader, a ResponseArtifact) error {
	if !a.MimetypeIs("application/json") {
		return nil
	}

	data := map[string]string{}

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&data); err != nil {
		return nil //  JSON decoder error, ignore this artifact
	}

	if len(data) != 1 {
		return nil // Not our JSON, ignore this artifact
	}

	if _, ok := data["nonce"]; !ok {
		return nil // Not our JSON, ignore this artifact
	}

	// We have only one JSON field called "nonce"

	a.SetArtifactMetadata(ArtifactMetadata{
		Type:        mimetypes.SnapNonce,
		Name:        "nonce",
		Description: fmt.Sprintf("Snap nonce JSON response"),
	})

	a.SetResponseApproved(ins, "valid snap nonce")

	return nil
}
