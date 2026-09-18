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

func needSkipLink(r *Renderer, dest []byte) bool {
	if r.Opts.Flags&SkipLinks != 0 {
		return true
	}
	isSafeURL := r.IsSafeURLOverride
	if isSafeURL == nil {
		isSafeURL = parser.IsSafeURL
	}
	return r.Opts.Flags&Safelink != 0 && !isSafeURL(dest) && !bytes.HasPrefix(dest, []byte("mailto:"))
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
