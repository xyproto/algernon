libdns - Universal DNS provider APIs for Go
===========================================

<a href="https://pkg.go.dev/github.com/libdns/libdns"><img src="https://img.shields.io/badge/godoc-reference-blue.svg"></a>

**⚠️ Work-in-progress. Exported APIs are subject to change.**

`libdns` is a collection of free-range DNS provider client implementations written in Go! With libdns packages, your Go program can manage DNS records across any supported providers. A "provider" is a service or program that manages a DNS zone.

This repository defines the core APIs that provider packages should implement. They are small and idiomatic Go interfaces with well-defined semantics for managing DNS records.

The interfaces include:

- [`RecordGetter`](https://pkg.go.dev/github.com/libdns/libdns#RecordGetter) to list records.
- [`RecordAppender`](https://pkg.go.dev/github.com/libdns/libdns#RecordAppender) to create new records.
- [`RecordSetter`](https://pkg.go.dev/github.com/libdns/libdns#RecordSetter) to set (create or update) records.
- [`RecordDeleter`](https://pkg.go.dev/github.com/libdns/libdns#RecordDeleter) to delete records.
- [`ZoneLister`](https://pkg.go.dev/github.com/libdns/libdns#ZoneLister) to list zones.

[See full godoc for detailed documentation.](https://pkg.go.dev/github.com/libdns/libdns)


## Example

To work with DNS records managed by Cloudflare, for example, we can use [libdns/cloudflare](https://pkg.go.dev/github.com/libdns/cloudflare):

```go
import (
	"github.com/libdns/cloudflare"
	"github.com/libdns/libdns"
)

ctx := context.TODO()

zone := "example.com."

// configure the DNS provider (choose any from github.com/libdns)
provider := cloudflare.Provider{APIToken: "topsecret"}

// list records
recs, err := provider.GetRecords(ctx, zone)

// create records (AppendRecords is similar, with different semantics)
newRecs, err := provider.SetRecords(ctx, zone, []libdns.Record{
	libdns.Address{
		Name:  "@",
		Value: netip.MustParseAddr("1.2.3.4"),
	},
})

// delete records
deletedRecs, err := provider.DeleteRecords(ctx, zone, []libdns.Record{
	libdns.TXT{
		Name: "subdomain",
		Text: "txt value I want to delete"
	},
})

// no matter which provider you use, the code stays the same!
// (some providers have caveats; see their package documentation)
```


## Implementing new provider packages

Provider packages are 100% written and maintained by the community! Collectively, we as members of the community each maintain the packages for providers we personally use.

**[Instructions for adding new libdns packages](https://github.com/libdns/libdns/wiki/Implementing-a-libdns-package)** are on this repo's wiki. Please feel free to contribute yours!


## Similar projects

**[OctoDNS](https://github.com/github/octodns)** is a suite of tools written in Python for managing DNS. However, its approach is a bit heavy-handed when all you need are small, incremental changes to a zone:

> WARNING: OctoDNS assumes ownership of any domain you point it to. When you tell it to act it will do whatever is necessary to try and match up states including deleting any unexpected records. Be careful when playing around with OctoDNS. 

This is incredibly useful when you are maintaining your own zone file, but risky when you just need incremental changes.

**[StackExchange/dnscontrol](https://github.com/StackExchange/dnscontrol)** is written in Go, but is similar to OctoDNS in that it tends to obliterate your entire zone and replace it with your input. Again, this is very useful if you are maintaining your own master list of records, but doesn't do well for simply adding or removing records.

**[go-acme/lego](https://github.com/go-acme/lego)** supports many DNS providers, but their APIs are only capable of setting and deleting TXT records for ACME challenges.

**[miekg/dns](https://github.com/miekg/dns)** is a comprehensive, low-level DNS library for Go programs. It is well-maintained and extremely thorough, but also too low-level to be productive for our use cases.

**`libdns`** takes inspiration from the above projects but aims for a more generally-useful set of high-level APIs that homogenize pretty well across providers. In contrast to the above projects, libdns can add, set, delete, and get arbitrary records from a zone without obliterating it (although syncing up an entire zone is also possible!). Its APIs also include context so long-running calls can be cancelled early, for example to accommodate on-line config changes downstream. libdns interfaces are also smaller and more composable. Additionally, libdns can grow to support a nearly infinite number of DNS providers without added bloat, because each provider implementation is a separate Go module, which keeps your builds lean and fast.

In summary, the goal is that libdns providers can do what the above libraries/tools can do, but with more flexibility: they can create and delete TXT records for ACME challenges, they can replace entire zones, but they can also do incremental changes or simply read records.

**Whatever libdns is used for with your DNS zone, it is presumed that only your libdns code is manipulating that (part of your) zone.** This package does not provide synchronization primitives, but your own code can do that if necessary.


## Record abstraction

How records are represented across providers varies widely, and each kind of record has different fields and semantics.

Realistically, libdns should enable most common record manipulations, but may not be able to fit absolutely 100% of all possibilities with DNS in a provider-agnostic way. That is probably OK; and given the wide varieties in DNS record types and provider APIs, it would be unreasonable to expect otherwise. Our goal is 100% fulfillment of ~99% of use cases / user requirements, not 100% fulfillment of 100% of use cases.
