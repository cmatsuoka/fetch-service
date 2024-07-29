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

package gomod

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	. "github.com/canonical/fetch-service/inspectors/common"
)

type ImportRedirectorInspector struct {
}

func NewImportRedirectorInspector() *ImportRedirectorInspector {
	return &ImportRedirectorInspector{}
}

func (ImportRedirectorInspector) ID() string {
	return "go.import-redirector"
}

func (ins *ImportRedirectorInspector) InspectRequest(a RequestArtefact) error {
	u, err := url.Parse(a.DownloadURL())
	if err != nil {
		return fmt.Errorf("cannot parse URL: %s", err)
	}

	if u.Query().Get("go-get") != "1" {
		return nil
	}

	_, err = newImportRedirectorUrlInfo(u)
	if err != nil {
		return nil // we don't recognize this request
	}

	a.SetRequestPending(ins, "request matches valid URL")
	return nil
}

func (ins *ImportRedirectorInspector) InspectArtefact(f ArtefactReader, a ResponseArtefact) error {
	if !a.MimetypeIs("text/html") {
		return nil
	}

	z := html.NewTokenizer(f)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if err := z.Err(); err != io.EOF {
				return err
			}
			return nil // end of file

		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			if t.Data == "meta" {
				imp, ok := extractMetaProperty(t, "go-import")
				if ok {
					u, err := url.Parse(a.DownloadURL())
					if err != nil {
						return err
					}

					var host = u.Host
					if vidx := strings.IndexByte(host, ':'); vidx > 0 {
						host = host[:vidx]
					}

					a.SetArtefactMetadata(ArtefactMetadata{
						Type:        "text/html",
						Name:        "Go import redirector",
						Description: "HTML file with go-import meta tag",
						Vendor:      host,
					})

					a.SetResponseApproved(ins, "document contains go-import meta tag").Annotate(
						Annotation{
							"go-import": imp,
						},
					)
					return nil
				}
			}
		}
	}
}

func extractMetaProperty(t html.Token, name string) (content string, ok bool) {
	for _, attr := range t.Attr {
		if attr.Key == "name" && attr.Val == name {
			ok = true
		}

		if attr.Key == "content" {
			content = attr.Val
		}
	}

	return
}
