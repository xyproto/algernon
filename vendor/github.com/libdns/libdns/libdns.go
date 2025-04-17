// Package [libdns] defines core interfaces that should be implemented by
// packages that interact with DNS provider clients. These interfaces are
// small and idiomatic Go interfaces with well-defined semantics for the
// purposes of reading and manipulating DNS records using DNS provider APIs.
//
// This documentation uses the definitions for terms from RFC 9499:
// https://datatracker.ietf.org/doc/html/rfc9499
//
// This package represents DNS records in two primary ways: as opaque [RR]
// structs, where the data is serialized as a single string as in a zone file;
// and as individual type structures, where the data is parsed into its separate
// fields for easier manipulation by Go programs (for example: [SRV] and [HTTPS]
// types). This hybrid design offers great flexibility for both DNS provider
// packages and consumer Go programs.
//
// This package represents records flexibly with the [Record] interface, which
// is any type that can transform itself into the [RR] struct, which is a
// type-agnostic [Resource Record] (that is, a name, type, class, TTL, and data).
// Specific record types such as [Address], [SRV], [TXT], and others implement
// the [Record] interface.
//
// Implementations of the libdns interfaces should accept as input any [Record]
// value, and should return as output the concrete struct types that implement
// the [Record] interface (i.e. [Address], [TXT], [ServiceBinding], etc). This
// is important to ensure the provider libraries are robust and also predictable:
// callers can reliably type-switch on the output to immediately access structured
// data about each record without the possibility of errors. Returned values should
// be of types defined by this package to make type-assertions reliable.
//
// Records are described independently of any particular zone, a convention that
// grants records portability across zones. As such, record names are partially
// qualified, i.e. relative to the zone. For example, a record called “sub” in
// zone “example.com.” represents a fully-qualified domain name (FQDN) of
// “sub.example.com.”. Implementations should expect that input records conform
// to this standard, while also ensuring that output records do; adjustments to
// record names may need to be made before or after provider API calls, for example,
// to maintain consistency with all other [libdns] packages. Helper functions are
// available in this package to convert between relative and absolute names;
// see [RelativeName] and [AbsoluteName].
//
// Although zone names are a required input, [libdns] does not coerce any
// particular representation of DNS zones; only records. Since zone name and
// records are separate inputs in [libdns] interfaces, it is up to the caller to
// maintain the pairing between a zone's name and its records.
//
// All interface implementations must be safe for concurrent/parallel use,
// meaning 1) no data races, and 2) simultaneous method calls must result in
// either both their expected outcomes or an error. For example, if
// [libdns.RecordAppender.AppendRecords] is called simultaneously, and two API
// requests are made to the provider at the same time, the result of both requests
// must be visible after they both complete; if the provider does not synchronize
// the writing of the zone file and one request overwrites the other, then the
// client implementation must take care to synchronize on behalf of the incompetent
// provider. This synchronization need not be global; for example: the scope of
// synchronization might only need to be within the same zone, allowing multiple
// requests at once as long as all of them are for different zone. (Exact logic
// depends on the provider.)
//
// Some service providers APIs may enforce rate limits or have sporadic errors.
// It is generally expected that libdns provider packages implement basic retry
// logic (e.g. retry up to 3-5 times with backoff in the event of a connection error
// or some HTTP error that may be recoverable, including 5xx or 429s) when it is
// safe to do so. Retrying/recovering from errors should not add substantial latency,
// though. If it will take longer than a couple seconds, best to return an error.
//
// [Resource Record]: https://en.wikipedia.org/wiki/Domain_Name_System#Resource_records
package libdns

import (
	"context"
	"strings"
)

// [RecordGetter] can get records from a DNS zone.
type RecordGetter interface {
	// GetRecords returns all the records in the DNS zone.
	//
	// DNSSEC-related records are typically not included in the output, but this
	// behavior is implementation-defined. If an implementation includes DNSSEC
	// records in the output, this behavior should be documented.
	//
	// Implementations must honor context cancellation and be safe for concurrent
	// use.
	GetRecords(ctx context.Context, zone string) ([]Record, error)
}

// [RecordAppender] can non-destructively add new records to a DNS zone.
type RecordAppender interface {
	// AppendRecords creates the inputted records in the given zone and returns
	// the populated records that were created. It never changes existing records.
	//
	// Therefore, it makes little sense to use this method with CNAME-type
	// records since if there are no existing records with the same name, it
	// behaves the same as [libdns.RecordSetter.SetRecords], and if there are
	// existing records with the same name, it will either fail or leave the
	// zone in an invalid state.
	//
	// Implementations should return struct types defined by this package which
	// correspond with the specific RR-type, rather than the [RR] struct, if possible.
	//
	// Implementations must honor context cancellation and be safe for concurrent
	// use.
	AppendRecords(ctx context.Context, zone string, recs []Record) ([]Record, error)
}

// [RecordSetter] can set new or update existing records in a DNS zone.
type RecordSetter interface {
	// SetRecords updates the zone so that the records described in the input are
	// reflected in the output. It may create or update records or—depending on
	// the record type—delete records to maintain parity with the input. No other
	// records are affected. It returns the records which were set.
	//
	// For any (name, type) pair in the input, SetRecords ensures that the only
	// records in the output zone with that (name, type) pair are those that were
	// provided in the input.
	//
	// In RFC 9499 terms, SetRecords appends, modifies, or deletes records in the
	// zone so that for each RRset in the input, the records provided in the input
	// are the only members of their RRset in the output zone.
	//
	// Implementations may decide whether or not to support DNSSEC-related records
	// in calls to SetRecords, but should document their decision. Note that the
	// decision to support DNSSEC records in SetRecords is independent of the
	// decision to support them in [libdns.RecordGetter.GetRecords], so callers
	// should not blindly call SetRecords with the output of
	// [libdns.RecordGetter.GetRecords].
	//
	// Calls to SetRecords are presumed to be atomic; that is, if err == nil,
	// then all of the requested changes were made; if err != nil, then none of
	// the requested changes were made, and the zone is as if the method was
	// never called. Some provider APIs may not support atomic operations, so it
	// is recommended that implementations synthesize atomicity by transparently
	// rolling back changes on failure; if this is not possible, then it should
	// be clearly documented that errors may result in partial changes to the
	// zone.
	//
	// If SetRecords is used to add a CNAME record to a name with other existing
	// non-DNSSEC records, implementations may either fail with an error, add
	// the CNAME and leave the other records in place (in violation of the DNS
	// standards), or add the CNAME and remove the other preexisting records.
	// Therefore, users should proceed with caution when using SetRecords with
	// CNAME records.
	//
	// Implementations should return struct types defined by this package which
	// correspond with the specific RR-type, rather than the [RR] struct, if possible.
	//
	// Implementations must honor context cancellation and be safe for concurrent
	// use.
	//
	// # Examples
	//
	// Example 1:
	//
	//	;; Original zone
	//	example.com. 3600 IN A   192.0.2.1
	//	example.com. 3600 IN A   192.0.2.2
	//	example.com. 3600 IN TXT "hello world"
	//
	//	;; Input
	//	example.com. 3600 IN A   192.0.2.3
	//
	//	;; Resultant zone
	//	example.com. 3600 IN A   192.0.2.3
	//	example.com. 3600 IN TXT "hello world"
	//
	// Example 2:
	//
	//	;; Original zone
	//	a.example.com. 3600 IN AAAA 2001:db8::1
	//	a.example.com. 3600 IN AAAA 2001:db8::2
	//	b.example.com. 3600 IN AAAA 2001:db8::3
	//	b.example.com. 3600 IN AAAA 2001:db8::4
	//
	//	;; Input
	//	a.example.com. 3600 IN AAAA 2001:db8::1
	//	a.example.com. 3600 IN AAAA 2001:db8::2
	//	a.example.com. 3600 IN AAAA 2001:db8::5
	//
	//	;; Resultant zone
	//	a.example.com. 3600 IN AAAA 2001:db8::1
	//	a.example.com. 3600 IN AAAA 2001:db8::2
	//	a.example.com. 3600 IN AAAA 2001:db8::5
	//	b.example.com. 3600 IN AAAA 2001:db8::3
	//	b.example.com. 3600 IN AAAA 2001:db8::4
	SetRecords(ctx context.Context, zone string, recs []Record) ([]Record, error)
}

// [RecordDeleter] can delete records from a DNS zone.
type RecordDeleter interface {
	// DeleteRecords deletes the given records from the zone if they exist in the
	// zone and exactly match the input. If the input records do not exist in the
	// zone, they are silently ignored. DeleteRecords returns only the the records
	// that were deleted, and does not return any records that were provided in the
	// input but did not exist in the zone.
	//
	// DeleteRecords only deletes records from the zone that *exactly* match the
	// input records—that is, the name, type, TTL, and value all must be identical
	// to a record in the zone for it to be deleted.
	//
	// As a special case, you may leave any of the fields [libdns.Record.Type],
	// [libdns.Record.TTL], or [libdns.Record.Value] empty ("", 0, and ""
	// respectively). In this case, DeleteRecords will delete any records that
	// match the other fields, regardless of the value of the fields that were left
	// empty. Note that this behavior does *not* apply to the [libdns.Record.Name]
	// field, which must always be specified.
	//
	// Note that it is semantically invalid to remove the last “NS” record from a
	// zone, so attempting to do is undefined behavior.
	//
	// Implementations should return struct types defined by this package which
	// correspond with the specific RR-type, rather than the [RR] struct, if possible.
	//
	// Implementations must honor context cancellation and be safe for concurrent
	// use.
	DeleteRecords(ctx context.Context, zone string, recs []Record) ([]Record, error)
}

// [ZoneLister] can list available DNS zones.
type ZoneLister interface {
	// ListZones returns the list of available DNS zones for use by other
	// [libdns] methods. Not every upstream provider API supports listing
	// available zones, and very few [libdns]-dependent packages use this
	// method, so this method is optional.
	//
	// Implementations should return struct types defined by this package which
	// correspond with the specific RR-type, rather than the [RR] struct, if possible.
	//
	// Implementations must honor context cancellation and be safe for
	// concurrent use.
	ListZones(ctx context.Context) ([]Zone, error)
}

// [Zone] is a generalized representation of a DNS zone.
type Zone struct {
	Name string
}

// [RelativeName] makes “fqdn” relative to “zone”. For example, for a FQDN of
// “sub.example.com” and a zone of “example.com.”, it returns “sub”.
//
// If fqdn is the same as zone (and both are non-empty), “@” is returned.
//
// If fqdn cannot be expressed relative to zone, the input fqdn is
// returned.
func RelativeName(fqdn, zone string) string {
	// liberally ignore trailing dots on both fqdn and zone, because
	// the relative name won't have a trailing dot anyway; I assume
	// this won't be problematic...?
	// (initially implemented because Cloudflare returns "fully-
	// qualified" domains in their records without a trailing dot,
	// but the input zone typically has a trailing dot)
	rel := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(fqdn, "."), strings.TrimSuffix(zone, ".")), ".")
	if rel == "" && fqdn != "" && zone != "" {
		return "@"
	}
	return rel
}

// [AbsoluteName] makes name into a fully-qualified domain name (FQDN) by
// prepending it to zone and tidying up the dots. For example, an input of
// name “sub” and zone “example.com.” will return “sub.example.com.”. If
// the name ends with a dot, it will be returned as the FQDN.
//
// Using “@” as the name is the recommended way to represent the root of the
// zone; however, unlike the [Record] struct, using the empty string "" for the
// name *is* permitted here, and will be treated identically to “@”.
//
// In the name already has a trailing dot, it is returned as-is. This is similar
// to the behavior of [path/filepath.Abs], and means that [AbsoluteName] is
// idempotent, so it is safe to call multiple times without first checking if
// the name is absolute or relative.
func AbsoluteName(name, zone string) string {
	if zone == "" {
		return strings.Trim(name, ".")
	}
	if name == "" || name == "@" {
		return zone
	}
	if strings.HasSuffix(name, ".") {
		// Already a FQDN, so just return it
		return name
	}
	return name + "." + zone
}
