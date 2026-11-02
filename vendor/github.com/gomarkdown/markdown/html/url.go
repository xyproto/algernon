package html

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown/parser"
)

func isRelativeLink(link []byte) bool {
	if len(link) == 0 {
		return true
	}
	switch link[0] {
	case '#':
		return true
	case '/':
		return len(link) == 1 || link[1] != '/'
	case '.':
		return bytes.HasPrefix(link, []byte("./")) || bytes.HasPrefix(link, []byte("../"))
	}
	return false
}

func isBareRelativePath(link []byte) bool {
	return len(link) > 0 && link[0] != '#' && link[0] != '/' &&
		!bytes.HasPrefix(link, []byte("./")) &&
		!bytes.HasPrefix(link, []byte("../")) &&
		!bytes.ContainsRune(link, ':')
}

func AddAbsPrefix(link []byte, prefix string) []byte {
	if len(link) == 0 || prefix == "" || !isRelativeLink(link) || link[0] == '.' {
		return link
	}
	if link[0] != '/' {
		prefix += "/"
	}
	return []byte(prefix + string(link))
}

func AddAbsPrefixToImage(link []byte, prefix string) []byte {
	if len(link) == 0 || prefix == "" {
		return link
	}
	if !isBareRelativePath(link) {
		return AddAbsPrefix(link, prefix)
	}
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	return []byte(prefix + string(link))
}

func appendLinkAttrs(attrs []string, flags Flags, link []byte) []string {
	if isRelativeLink(link) {
		return attrs
	}
	rels := make([]string, 0, 3)
	for _, rel := range []struct {
		flag Flags
		name string
	}{
		{NofollowLinks, "nofollow"},
		{NoreferrerLinks, "noreferrer"},
		{NoopenerLinks, "noopener"},
	} {
		if flags&rel.flag != 0 {
			rels = append(rels, rel.name)
		}
	}
	if flags&HrefTargetBlank != 0 {
		attrs = append(attrs, `target="_blank"`)
	}
	if len(rels) > 0 {
		attrs = append(attrs, fmt.Sprintf("rel=%q", strings.Join(rels, " ")))
	}
	return attrs
}

// outputDest returns a link or image destination as it will be written to the
// output, before HTML escaping. Safety checks must look at this value so they
// judge the same URL the browser gets. The CommonMark parser has already
// decoded character references.
func (r *Renderer) outputDest(dest []byte) []byte {
	if r.commonMark {
		return dest
	}
	return unescapeEntities(dest)
}

func (r *Renderer) isSafeURL(dest []byte) bool {
	if r.IsSafeURLOverride != nil {
		return r.IsSafeURLOverride(dest)
	}
	return parser.IsSafeURL(dest)
}

func needSkipLink(r *Renderer, dest []byte) bool {
	if r.Opts.Flags&SkipLinks != 0 {
		return true
	}
	dest = r.outputDest(dest)
	return r.Opts.Flags&Safelink != 0 && !r.isSafeURL(dest) && !bytes.HasPrefix(dest, []byte("mailto:"))
}

// needSkipImage reports whether Safelink rejects an image source. Images also
// accept what is normal for a source but not for IsSafeURL: bare relative paths
// such as foo.png and inline raster images.
func needSkipImage(r *Renderer, dest []byte) bool {
	if r.Opts.Flags&Safelink == 0 {
		return false
	}
	dest = r.outputDest(dest)
	return !r.isSafeURL(dest) && !isSchemelessPath(dest) && !isDataRasterImage(dest)
}

// normalizeURLBytes removes what browsers ignore when parsing a URL: leading
// control characters and spaces, and any tab or newline. Otherwise a tab inside
// "javascript:" could hide the scheme.
func normalizeURLBytes(url []byte) []byte {
	url = bytes.TrimLeftFunc(url, func(c rune) bool { return c <= ' ' })
	if bytes.IndexAny(url, "\t\r\n") < 0 {
		return url
	}
	return bytes.Map(func(c rune) rune {
		if c == '\t' || c == '\r' || c == '\n' {
			return -1
		}
		return c
	}, url)
}

var slashes = []byte(`/\`)

// isSchemelessPath reports whether url stays on the current scheme and host.
// Browsers treat a backslash like a slash, so /\evil.com is another host.
func isSchemelessPath(url []byte) bool {
	url = normalizeURLBytes(url)
	if len(url) >= 2 && bytes.IndexByte(slashes, url[0]) >= 0 && bytes.IndexByte(slashes, url[1]) >= 0 {
		return false
	}
	i := bytes.IndexAny(url, `:/\?#`)
	return i < 0 || url[i] != ':'
}

func isDataRasterImage(url []byte) bool {
	url = bytes.ToLower(normalizeURLBytes(url))
	for _, mime := range []string{"png", "gif", "jpeg", "jpg", "webp"} {
		rest, ok := bytes.CutPrefix(url, []byte("data:image/"+mime))
		if ok && len(rest) > 0 && (rest[0] == ';' || rest[0] == ',') {
			return true
		}
	}
	return false
}

func appendLanguageAttr(attrs []string, info []byte) []string {
	if len(info) == 0 {
		return attrs
	}
	if end := bytes.IndexAny(info, "\t "); end >= 0 {
		info = info[:end]
	}
	var class bytes.Buffer
	class.WriteString(`class="language-`)
	EscapeHTML(&class, info)
	class.WriteByte('"')
	return append(attrs, class.String())
}
