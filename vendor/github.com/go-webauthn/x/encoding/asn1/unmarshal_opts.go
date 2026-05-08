package asn1

type unmarshalOpts struct {
	allowTypeGeneralString bool
	allowBERIntegers       bool
}

// UnmarshalOpt describes a functional option for unmarshalling.
type UnmarshalOpt func(opts *unmarshalOpts)

// WithUnmarshalAllowTypeGeneralString allows the use of ASN.1 DER GeneralString type. This is an option since it
// deviates from stdlib.
func WithUnmarshalAllowTypeGeneralString(value bool) UnmarshalOpt {
	return func(opts *unmarshalOpts) {
		opts.allowTypeGeneralString = value
	}
}

// WithUnmarshalAllowBERIntegers permits the use of ASN.1 BER integer types. This is an option since it deviates from
// stdlib.
func WithUnmarshalAllowBERIntegers(value bool) UnmarshalOpt {
	return func(opts *unmarshalOpts) {
		opts.allowBERIntegers = value
	}
}
