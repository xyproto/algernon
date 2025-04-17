package libdns

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// Record is any type that can reduce itself to the [RR] struct.
//
// Primitive equality (“==”) between any two [Record]s is explicitly undefined;
// if implementations need to compare records, they should either define their
// own equality functions or compare the [RR] structs directly.
type Record interface {
	RR() RR
}

// RR represents a [DNS Resource Record], which resembles how records are
// represented by DNS servers in zone files.
//
// The fields in this struct are common to all RRs, with the data field
// being opaque; it has no particular meaning until it is parsed.
//
// This type should NOT be returned by implementations of the libdns interfaces;
// in other words, methods such as GetRecords, AppendRecords, etc., should
// not return RR values. Instead, they should return the structs corresponding
// to the specific RR types (such as [Address], [TXT], etc). This provides
// consistency for callers who can then reliably type-switch or type-assert the
// output without the possibility for errors.
//
// [DNS Resource Record]: https://en.wikipedia.org/wiki/Domain_Name_System#Resource_records
type RR struct {
	// The name of the record. It is partially qualified, relative to the zone.
	// For the sake of consistency, use "@" to represent the root of the zone.
	// An empty name typically refers to the last-specified name in the zone
	// file, which is only determinable in specific contexts.
	//
	// (For the following examples, assume the zone is “example.com.”)
	//
	// Examples:
	//   - “www” (for “www.example.com.”)
	//   - “@” (for “example.com.”)
	//   - “subdomain” (for “subdomain.example.com.”)
	//   - “sub.subdomain” (for “sub.subdomain.example.com.”)
	//
	// Invalid:
	//   - “www.example.com.” (fully-qualified)
	//   - “example.net.” (fully-qualified)
	//   - "" (empty)
	//
	// Valid, but probably doesn't do what you want:
	//   - “www.example.net” (refers to “www.example.net.example.com.”)
	Name string

	// The time-to-live of the record. This is represented in the DNS zone file as
	// an unsigned integral number of seconds, but is provided here as a
	// [time.Duration] for ease of use in Go code. Fractions of seconds will be
	// rounded down (truncated). A value of 0 means that the record should not be
	// cached. Some provider implementations may assume a default TTL from 0; to
	// avoid this, set TTL to a sub-second duration.
	//
	// Note that some providers may reject or silently increase TTLs that are below
	// a certain threshold, and that DNS resolvers may choose to ignore your TTL
	// settings, so it is recommended to not rely on the exact TTL value.
	TTL time.Duration

	// The type of the record as an uppercase string. DNS provider packages are
	// encouraged to support as many of the most common record types as possible,
	// especially: A, AAAA, CNAME, TXT, HTTPS, and SRV.
	//
	// Other custom record types may be supported with implementation-defined
	// behavior.
	Type string

	// The data (or "value") of the record. This field should be formatted in
	// the *unescaped* standard zone file syntax (technically, the "RDATA" field
	// as defined by RFC 1035 §5.1). Due to variances in escape sequences and
	// provider support, this field should not contain escapes. More concretely,
	// the following [libdns.Record]s
	//
	//  []libdns.TXT{
	//      {
	//          Name: "alpha",
	//          Text: `quotes " backslashes \000`,
	//      }, {
	//          Name: "beta",
	//          Text: "del: \x7F",
	//      },
	//  }
	//
	// should be equivalent to the following in zone file syntax:
	//
	//  alpha  0  IN  TXT  "quotes \" backslashes \\000"
	//  beta   0  IN  TXT  "del: \177"
	//
	// Implementations are not expected to support RFC 3597 “\#” escape
	// sequences, but may choose to do so if they wish.
	Data string
}

// RR returns itself. This may be the case when trying to parse an RR type
// that is not (yet) supported/implemented by this package.
func (r RR) RR() RR { return r }

// Parse returns a type-specific structure for this RR, if it is
// a known/supported type. Otherwise, it returns itself.
//
// Callers will typically want to type-assert (or use a type switch on)
// the return value to extract values or manipulate it.
func (r RR) Parse() (Record, error) {
	switch r.Type {
	case "A", "AAAA":
		return r.toAddress()
	case "CAA":
		return r.toCAA()
	case "CNAME":
		return r.toCNAME()
	case "HTTPS", "SVCB":
		return r.toServiceBinding()
	case "MX":
		return r.toMX()
	case "NS":
		return r.toNS()
	case "SRV":
		return r.toSRV()
	case "TXT":
		return r.toTXT()
	default:
		return r, nil
	}
}

func (r RR) toAddress() (Address, error) {
	if r.Type != "A" && r.Type != "AAAA" {
		return Address{}, fmt.Errorf("record type not A or AAAA: %s", r.Type)
	}

	ip, err := netip.ParseAddr(r.Data)
	if err != nil {
		return Address{}, fmt.Errorf("invalid IP address %q: %v", r.Data, err)
	}

	return Address{
		Name: r.Name,
		IP:   ip,
		TTL:  r.TTL,
	}, nil
}

func (r RR) toCAA() (CAA, error) {
	if expectedType := "CAA"; r.Type != expectedType {
		return CAA{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}

	fields := strings.Fields(r.Data)
	if expectedLen := 3; len(fields) != expectedLen {
		return CAA{}, fmt.Errorf(`malformed CAA value; expected %d fields in the form 'flags tag "value"'`, expectedLen)
	}

	flags, err := strconv.ParseUint(fields[0], 10, 8)
	if err != nil {
		return CAA{}, fmt.Errorf("invalid flags %s: %v", fields[0], err)
	}
	tag := fields[1]
	value := strings.Trim(fields[2], `"`)

	return CAA{
		Name:  r.Name,
		TTL:   r.TTL,
		Flags: uint8(flags),
		Tag:   tag,
		Value: value,
	}, nil
}

func (r RR) toCNAME() (CNAME, error) {
	if expectedType := "CNAME"; r.Type != expectedType {
		return CNAME{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}
	return CNAME{
		Name:   r.Name,
		TTL:    r.TTL,
		Target: r.Data,
	}, nil
}

func (r RR) toMX() (MX, error) {
	if expectedType := "MX"; r.Type != expectedType {
		return MX{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}

	fields := strings.Fields(r.Data)
	if expectedLen := 2; len(fields) != expectedLen {
		return MX{}, fmt.Errorf("malformed MX value; expected %d fields in the form 'preference target'", expectedLen)
	}

	priority, err := strconv.ParseUint(fields[0], 10, 16)
	if err != nil {
		return MX{}, fmt.Errorf("invalid priority %s: %v", fields[0], err)
	}
	target := fields[1]

	return MX{
		Name:       r.Name,
		TTL:        r.TTL,
		Preference: uint16(priority),
		Target:     target,
	}, nil
}

func (r RR) toNS() (NS, error) {
	if expectedType := "NS"; r.Type != expectedType {
		return NS{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}
	return NS{
		Name:   r.Name,
		TTL:    r.TTL,
		Target: r.Data,
	}, nil
}

func (r RR) toSRV() (SRV, error) {
	if expectedType := "SRV"; r.Type != expectedType {
		return SRV{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}

	fields := strings.Fields(r.Data)
	if expectedLen := 4; len(fields) != expectedLen {
		return SRV{}, fmt.Errorf("malformed SRV value; expected %d fields in the form 'priority weight port target'", expectedLen)
	}

	priority, err := strconv.ParseUint(fields[0], 10, 16)
	if err != nil {
		return SRV{}, fmt.Errorf("invalid priority %s: %v", fields[0], err)
	}
	weight, err := strconv.ParseUint(fields[1], 10, 16)
	if err != nil {
		return SRV{}, fmt.Errorf("invalid weight %s: %v", fields[0], err)
	}
	port, err := strconv.ParseUint(fields[2], 10, 16)
	if err != nil {
		return SRV{}, fmt.Errorf("invalid port %s: %v", fields[0], err)
	}
	target := fields[3]

	parts := strings.SplitN(r.Name, ".", 3)
	if len(parts) < 3 {
		return SRV{}, fmt.Errorf("name %v does not contain enough fields; expected format: '_service._proto.name'", r.Name)
	}

	return SRV{
		Service:   strings.TrimPrefix(parts[0], "_"),
		Transport: strings.TrimPrefix(parts[1], "_"),
		Name:      parts[2],
		TTL:       r.TTL,
		Priority:  uint16(priority),
		Weight:    uint16(weight),
		Port:      uint16(port),
		Target:    target,
	}, nil
}

func (r RR) toServiceBinding() (ServiceBinding, error) {
	recType := r.Type
	if recType != "HTTPS" && recType != "SVCB" {
		return ServiceBinding{}, fmt.Errorf("record type not SVCB or HTTPS: %s", r.Type)
	}

	paramsParts := strings.SplitN(r.Data, " ", 3)
	if minParts := 2; len(paramsParts) < minParts { // SvcParams can be empty
		return ServiceBinding{}, fmt.Errorf("malformed HTTPS value; expected at least %d fields in the form 'priority target [SvcParams]'", minParts)
	}

	priority, err := strconv.ParseUint(strings.TrimSpace(paramsParts[0]), 10, 16)
	if err != nil {
		return ServiceBinding{}, fmt.Errorf("invalid priority %s: %v", paramsParts[0], err)
	}
	target := paramsParts[1]

	svcParams := SvcParams{}
	if len(paramsParts) > 2 {
		svcParams, err = ParseSvcParams(paramsParts[2])
		if err != nil {
			return ServiceBinding{}, fmt.Errorf("invalid SvcParams: %w", err)
		}
	}

	scheme := ""
	var port uint64 = 0
	nameParts := strings.SplitN(r.Name, ".", 3)
	if strings.HasPrefix(nameParts[0], "_") && strings.HasPrefix(nameParts[1], "_") {
		portStr := strings.TrimPrefix(nameParts[0], "_")
		scheme = strings.TrimPrefix(nameParts[1], "_")

		port, err = strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return ServiceBinding{}, fmt.Errorf("invalid port %s: %v", portStr, err)
		}
		nameParts = nameParts[2:]
	} else if strings.HasPrefix(nameParts[0], "_") {
		scheme = strings.TrimPrefix(nameParts[0], "_")
		nameParts = nameParts[1:]
	}

	if scheme == "" && recType == "HTTPS" {
		scheme = "https"
	} else if port > 0 && scheme == "https" && recType == "HTTPS" {
		// ok
	} else if scheme != "" && recType == "SVCB" {
		// ok
	} else {
		return ServiceBinding{}, fmt.Errorf("invalid name %q; expected format: '_port._proto.name' or '_proto.name'", r.Name)
	}

	return ServiceBinding{
		Scheme:        scheme,
		URLSchemePort: uint16(port),
		Name:          strings.Join(nameParts, "."),
		TTL:           r.TTL,
		Priority:      uint16(priority),
		Target:        target,
		Params:        svcParams,
	}, nil
}

func (r RR) toTXT() (TXT, error) {
	if expectedType := "TXT"; r.Type != expectedType {
		return TXT{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}
	return TXT{
		Name: r.Name,
		TTL:  r.TTL,
		Text: r.Data,
	}, nil
}

// SvcParams represents SvcParamKey=SvcParamValue pairs as described in
// RFC 9460 section 2.1. See https://www.rfc-editor.org/rfc/rfc9460#presentation.
//
// Note that this type is not primitively comparable, so using == for
// structs containnig a field of this type will panic.
type SvcParams map[string][]string

// String serializes svcParams into zone presentation format described by RFC 9460.
func (params SvcParams) String() string {
	var sb strings.Builder
	for key, vals := range params {
		if sb.Len() > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString(key)
		var hasVal, needsQuotes bool
		for _, val := range vals {
			if len(val) > 0 {
				hasVal = true
			}
			if strings.ContainsAny(val, `" `) {
				needsQuotes = true
			}
			if hasVal && needsQuotes {
				break
			}
		}
		if hasVal {
			sb.WriteRune('=')
		}
		if needsQuotes {
			sb.WriteRune('"')
		}
		for i, val := range vals {
			if i > 0 {
				sb.WriteRune(',')
			}
			val = strings.ReplaceAll(val, `"`, `\"`)
			val = strings.ReplaceAll(val, `,`, `\,`)
			sb.WriteString(val)
		}
		if needsQuotes {
			sb.WriteRune('"')
		}
	}
	return sb.String()
}

// ParseSvcParams parses a SvcParams string described by RFC 9460 into a structured type.
func ParseSvcParams(input string) (SvcParams, error) {
	input = strings.TrimSpace(input)
	if len(input) > 4096 {
		return nil, fmt.Errorf("input too long: %d", len(input))
	}
	params := make(SvcParams)
	if len(input) == 0 {
		return params, nil
	}

	// adding a space makes it easier to find the end of last key-value pair
	input += " "

	for cursor := 0; cursor < len(input); cursor++ {
		var key, rawVal string

	keyValPair:
		for i := cursor; i < len(input); i++ {
			switch input[i] {
			case '=':
				key = strings.ToLower(strings.TrimSpace(input[cursor:i]))
				i++
				cursor = i

				var quoted bool
				if input[cursor] == '"' {
					quoted = true
					i++
					cursor = i
				}

				var escaped bool

				for j := cursor; j < len(input); j++ {
					switch input[j] {
					case '"':
						if !quoted {
							return nil, fmt.Errorf("illegal DQUOTE at position %d", j)
						}
						if !escaped {
							// end of quoted value
							rawVal = input[cursor:j]
							j++
							cursor = j
							break keyValPair
						}
					case '\\':
						escaped = true
					case ' ', '\t', '\n', '\r':
						if !quoted {
							// end of unquoted value
							rawVal = input[cursor:j]
							cursor = j
							break keyValPair
						}
					default:
						escaped = false
					}
				}

			case ' ', '\t', '\n', '\r':
				// key with no value (flag)
				key = input[cursor:i]
				params[key] = []string{}
				cursor = i
				break keyValPair
			}
		}

		if rawVal == "" {
			continue
		}

		var sb strings.Builder

		var escape int // start of escape sequence (after \, so 0 is never a valid start)
		for i := 0; i < len(rawVal); i++ {
			ch := rawVal[i]
			if escape > 0 {
				// validate escape sequence
				// (RFC 9460 Appendix A)
				// escaped:   "\" ( non-digit / dec-octet )
				// non-digit: "%x21-2F / %x3A-7E"
				// dec-octet: "0-255 as a 3-digit decimal number"
				if ch >= '0' && ch <= '9' {
					// advance to end of decimal octet, which must be 3 digits
					i += 2
					if i > len(rawVal) {
						return nil, fmt.Errorf("value ends with incomplete escape sequence: %s", rawVal[escape:])
					}
					decOctet, err := strconv.Atoi(rawVal[escape : i+1])
					if err != nil {
						return nil, err
					}
					if decOctet < 0 || decOctet > 255 {
						return nil, fmt.Errorf("invalid decimal octet in escape sequence: %s (%d)", rawVal[escape:i], decOctet)
					}
					sb.WriteRune(rune(decOctet))
					escape = 0
					continue
				} else if (ch < 0x21 || ch > 0x2F) && (ch < 0x3A && ch > 0x7E) {
					return nil, fmt.Errorf("illegal escape sequence %s", rawVal[escape:i])
				}
			}
			switch ch {
			case ';', '(', ')':
				// RFC 9460 Appendix A:
				// > contiguous  = 1*( non-special / escaped )
				// > non-special is VCHAR minus DQUOTE, ";", "(", ")", and "\".
				return nil, fmt.Errorf("illegal character in value %q at position %d: %s", rawVal, i, string(ch))
			case '\\':
				escape = i + 1
			default:
				sb.WriteByte(ch)
				escape = 0
			}
		}

		params[key] = strings.Split(sb.String(), ",")
	}

	return params, nil
}
