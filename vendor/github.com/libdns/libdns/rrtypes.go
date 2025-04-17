package libdns

import (
	"fmt"
	"net/netip"
	"time"
)

// Address represents a parsed A-type or AAAA-type record,
// which associates a name with an IPv4 or IPv6 address
// respectively. This is typically how to "point a domain
// to your server."
//
// Since A and AAAA are semantically identical, with the
// exception of the bit length of the IP address in the
// data field, these record types are combined for ease of
// use in Go programs, which supports both address sizes,
// to help simplify code.
type Address struct {
	Name string
	TTL  time.Duration
	IP   netip.Addr
}

func (a Address) RR() RR {
	recType := "A"
	if a.IP.Is6() {
		recType = "AAAA"
	}
	return RR{
		Name: a.Name,
		TTL:  a.TTL,
		Type: recType,
		Data: a.IP.String(),
	}
}

// CAA represents a parsed CAA-type record, which is used to specify which PKIX
// certificate authorities are allowed to issue certificates for a domain. See
// also the [registry of flags and tags].
//
// [registry of flags and tags]: https://www.iana.org/assignments/caa-parameters/caa-parameters.xhtml
type CAA struct {
	Name  string
	TTL   time.Duration
	Flags uint8 // As of March 2025, the only valid values are 0 and 128.
	Tag   string
	Value string
}

func (c CAA) RR() RR {
	return RR{
		Name: c.Name,
		TTL:  c.TTL,
		Type: "CAA",
		Data: fmt.Sprintf(`%d %s %q`, c.Flags, c.Tag, c.Value),
	}
}

// CNAME represents a CNAME-type record, which delegates
// authority to other names.
type CNAME struct {
	Name   string
	TTL    time.Duration
	Target string
}

func (c CNAME) RR() RR {
	return RR{
		Name: c.Name,
		TTL:  c.TTL,
		Type: "CNAME",
		Data: c.Target,
	}
}

// MX represents a parsed MX-type record, which is used to specify the hostnames
// of the servers that accept mail for a domain.
type MX struct {
	Name       string
	TTL        time.Duration
	Preference uint16 // Lower values indicate that clients should prefer this server. This field is similar to the “Priority” field in SRV records.
	Target     string // The hostname of the mail server
}

func (m MX) RR() RR {
	return RR{
		Name: m.Name,
		TTL:  m.TTL,
		Type: "MX",
		Data: fmt.Sprintf("%d %s", m.Preference, m.Target),
	}
}

// NS represents a parsed NS-type record, which is used to specify the
// authoritative nameservers for a zone. It is strongly recommended to have at
// least two NS records for redundancy.
//
// Note that the NS records present at the root level of a zone must match those
// delegated to by the parent zone. This means that changing the NS records for
// the root of a registered domain won't have any effect unless you also update
// the NS records with the domain registrar.
//
// Also note that the DNS standards forbid removing the last NS record for a
// zone, so if you want to replace all NS records, you should add the new ones
// before removing the old ones.
type NS struct {
	Name   string
	TTL    time.Duration
	Target string
}

func (n NS) RR() RR {
	return RR{
		Name: n.Name,
		TTL:  n.TTL,
		Type: "NS",
		Data: n.Target,
	}
}

// SRV represents a parsed SRV-type record, which is used to
// manifest services or instances that provide services on a
// network.
//
// The serialization of this record type takes the form:
//
//	_service._proto.name. ttl IN SRV priority weight port target.
//
// Note that all fields are mandatory.
type SRV struct {
	// “Service” is the name of the service being offered, without the leading
	// underscore. The correct value for this field is defined by the service
	// that you are serving (and is typically registered with IANA). Some
	// examples include "sip", "xmpp", "ldap", "minecraft", "stun", "turn", etc.
	Service string

	// “Transport” is the name of the transport protocol used by the service,
	// without the leading underscore. This is almost always "tcp" or "udp", but
	// "sctp" and "dccp" are technically valid as well.
	//
	// Note that RFC 2782 defines this field as “Proto[col]”, but we're using
	// the updated name “Transport” from RFC 6335 in order to avoid confusion
	// with the similarly-named field in the SVCB record type.
	Transport string

	Name     string
	TTL      time.Duration
	Priority uint16 // Lower values indicate that clients should prefer this server
	Weight   uint16 // Higher values indicate that clients should prefer this server when choosing between targets with the same priority
	Port     uint16 // The port on which the service is running.
	Target   string // The hostname of the server providing the service, which must not point to a CNAME.
}

func (s SRV) RR() RR {
	var name string
	if s.Service == "" && s.Transport == "" {
		// If both “Service” and “Transport” are empty, then we'll assume that
		// “Name” is complete as-is. This is fairly dubious, but could happen
		// if a properly-underscored CNAME points at a SRV without underscores.
		name = s.Name
	} else {
		// Otherwise, we need to prepend the underscores to the name.
		name = fmt.Sprintf("_%s._%s.%s", s.Service, s.Transport, s.Name)
	}

	return RR{
		Name: name,
		TTL:  s.TTL,
		Type: "SRV",
		Data: fmt.Sprintf("%d %d %d %s", s.Priority, s.Weight, s.Port, s.Target),
	}
}

// ServiceBinding represents a parsed ServiceBinding-type record, which is used to provide the
// target and various key–value parameters for a service. HTTPS records are
// defined as a “ServiceBinding-Compatible RR Type”, which means that their data
// structures are identical to ServiceBinding records, albeit with a different type name
// and semantics.
//
// HTTPS-type records are  used to provide clients with information for
// establishing HTTPS connections to servers. It may include data about ALPN,
// ECH, IP hints, and more.
//
// Unlike the other RR types that are hostname-focused or service-focused, ServiceBinding
// (“Service Binding”) records are URL-focused. This distinction is generally
// irrelevant, but is important when disusing the port fields.
type ServiceBinding struct {
	// “Scheme” is the scheme of the URL used to access the service, or some
	// other protocol identifier registered with IANA. This field should not
	// contain a leading underscore.
	//
	// If the scheme is set to "https", then a HTTPS-type record will be
	// generated; for all other schemes, a SVCB-type record will be generated.
	// As defined in RFC 9460, the schemes "http", "wss", and "ws" also map to
	// HTTPS records.
	//
	// Note that if a new SVCB-compatible RR type is defined and specified as
	// mapping to a scheme, then [libdns] may automatically generate that type
	// instead of SVCB at some point in the future. It is expected that any RFC
	// that proposes such a new type will ensure that this does not cause any
	// backwards compatibility issues.
	Scheme string

	// Warning: This field almost certainly does not do what you expect, and
	// should typically be unset (or set to 0).
	//
	// “URLSchemePort” is the port number that is explictly specified in a URL
	// when accessing a service. This field does not affect the port number that
	// is actually used to access the service, and unlike with SRV records, it
	// must be unset if you are using the default port for the scheme.
	//
	// # Examples
	//
	// In the typical case, you would have the following URL:
	//
	//  https://example.com/
	//
	// and then the client would lookup the following records:
	//
	//  example.com.  60  IN  HTTPS  1  example.net.  alpn=h2,h3
	//  example.net.  60  IN  A      192.0.2.1
	//
	// and then the client would connect to 192.0.2.1:443. But if you had the
	// same URL but the following records:
	//
	//  example.com.  60  IN  HTTPS  1  example.net.  alpn=h2,h3 port=1111
	//  example.net.  60  IN  A      192.0.2.2
	//
	// then the client would connect to 192.0.2.2:1111. But if you had the
	// following URL:
	//
	//  https://example.com:2222/
	//
	// then the client would lookup the following records:
	//
	//  _2222._https.example.com.  60  IN  HTTPS  1  example.net.  alpn=h2,h3
	//  example.net.               60  IN  A      192.0.2.3
	//
	// and the client would connect to 192.0.2.3:2222. And if you had the same
	// URL but the following records:
	//
	//  _2222._https.example.com.  60  IN  HTTPS  1  example.net.  alpn=h2,h3 port=3333
	//  example.net.               60  IN  A      192.0.2.4
	//
	// then the client would connect to 192.0.2.4:3333.
	//
	// So the key things to note here are that:
	//
	//  - If you want to change the port that the client connects to, you need
	//    to set the “port=” value in the “Params” field, not the
	//    “URLSchemePort”.
	//
	//  - The client will never lookup the HTTPS record prefixed with the
	//    underscored default port, so you should only set “URLSchemePort” if
	//    you are explicitly using a non-default port in the URL.
	//
	//  - It is completely valid to set the “port=” value in the “Params” field
	//    to the default port for the scheme, but also completely unnecessary.
	//
	//  - The “URLSchemePort” field and the “port=” value in the “Params” field
	//    are completely independent, with one exception: if you set the
	//    “URLSchemePort” field to a non-default port and leave the “port=”
	//    value in the “Params” field unset, then the client will default to the
	//    value of the “URLSchemePort” field, and not to the default port for
	//    the scheme.
	URLSchemePort uint16

	Name string
	TTL  time.Duration

	// “Priority” is the priority of the service, with lower values indicating
	// that clients should prefer this service over others.
	//
	// Note that Priority==0 is a special case, and indicates that the record
	// is an “Alias” record. Alias records behave like CNAME records, but are
	// allowed at the root of a zone. When in Alias mode, the Params field
	// should be unset.
	Priority uint16

	// “Target” is the target of the service, which is typically a hostname or
	// an alias (CNAME or other SVCB record). If this field is set to a single
	// dot ".", then the target is the same as the name of the record (without
	// the underscore-prefixed components, of course).
	Target string

	// “Params” is a map of key–value pairs that are used to specify various
	// parameters for the service. The keys are typically registered with IANA,
	// and which keys are valid is service-dependent.
	// https://www.iana.org/assignments/dns-svcb/dns-svcb.xhtml
	//
	// Note that there is a key called “mandatory”, but this does not mean that
	// it is mandatory for you to set the listed keys. Instead, this means that
	// if a client does not understand all of the listed keys, then it must
	// ignore the entire record. This is similar to the “critical” flag in CAA
	// records.
	Params SvcParams
}

// RR converts the parsed record data to a generic [Record] struct.
//
// EXPERIMENTAL; subject to change or removal.
func (s ServiceBinding) RR() RR {
	var name string
	var recType string
	if s.Scheme == "https" || s.Scheme == "http" || s.Scheme == "wss" || s.Scheme == "ws" {
		recType = "HTTPS"
		name = s.Name
		if s.URLSchemePort == 443 || s.URLSchemePort == 80 {
			// Ok, we'll correct your mistake for you.
			s.URLSchemePort = 0
		}
	} else {
		recType = "SVCB"
		name = fmt.Sprintf("_%s.%s", s.Scheme, s.Name)
	}

	if s.URLSchemePort != 0 {
		name = fmt.Sprintf("_%d.%s", s.URLSchemePort, name)
	}

	var params string
	if s.Priority == 0 && len(s.Params) != 0 {
		// The SvcParams should be empty in AliasMode, so we'll fix that for
		// you.
		params = ""
	} else {
		params = s.Params.String()
	}

	return RR{
		Name: name,
		TTL:  s.TTL,
		Type: recType,
		Data: fmt.Sprintf("%d %s %s", s.Priority, s.Target, params),
	}
}

// TXT represents a parsed TXT-type record, which is used to
// add arbitrary text data to a name in a DNS zone. It is often
// used for email integrity (DKIM/SPF), site verification, ACME
// challenges, and more.
type TXT struct {
	Name string
	TTL  time.Duration

	// The “Text” field contains the arbitrary data associated with the TXT
	// record. The contents of this field should *not* be wrapped in quotes as
	// libdns implementations are expected to quote any fields as necessary. In
	// addition, as discussed in the description of [libdns.RR.Data], you should
	// not include any escaped characters in this field, as libdns will escape
	// them for you.
	//
	// In the zone file format and the DNS wire format, a single TXT record is
	// composed of one or more strings of no more than 255 bytes each ([RFC 1035
	// §3.3.14], [RFC 7208 §3.3]). We eschew those restrictions here, and
	// instead treat the entire TXT as a single, arbitrary-length string. libdns
	// implementations are therefore expected to handle this as required by
	// their respective DNS provider APIs. See the [DNSControl explainer] on
	// this for more information.
	//
	// [RFC 1035 §3.3.14]: https://datatracker.ietf.org/doc/html/rfc1035#section-3.3.14
	// [RFC 7208 §3.3]: https://datatracker.ietf.org/doc/html/rfc7208#section-3.3
	// [DNSControl explainer]: https://docs.dnscontrol.org/developer-info/opinions#opinion-8-txt-records-are-one-long-string
	Text string
}

func (t TXT) RR() RR {
	return RR{
		Name: t.Name,
		TTL:  t.TTL,
		Type: "TXT",
		Data: t.Text,
	}
}
