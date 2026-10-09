/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package uploadsign

import (
	"net/url"
	"path"
	"strings"
)

// StorageKey normalizes a stored object key (or path-style / signed URL) for
// DB identity. bucket is the raw configured bucket name (may be empty); it is
// trimmed here. Pure logic: no global dependency, callers pass the bucket.
func StorageKey(s, bucket string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if u, err := url.Parse(s); err == nil && u.Scheme != "" && u.Host != "" {
		s = u.Path
	}
	s = strings.TrimPrefix(s, "/upload/")
	s = strings.TrimPrefix(s, "/")
	// Path-style OSS URLs are /{bucket}/{key}; strip bucket when present.
	if b := strings.Trim(bucket, "/"); b != "" {
		if s == b {
			return ""
		}
		if strings.HasPrefix(s, b+"/") {
			s = strings.TrimPrefix(s, b+"/")
		}
	}
	if path.IsAbs(s) || strings.Contains(s, "..") || s != path.Clean(s) {
		return ""
	}
	return s
}
