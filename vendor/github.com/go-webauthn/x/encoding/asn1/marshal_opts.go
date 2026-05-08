package asn1

type marshalOpts struct {
	slicePreserveTypes bool
	sliceAllowStrings  bool
}

// MarshalOpt describes a functional option for marshalling.
type MarshalOpt func(opts *marshalOpts)

// WithMarshalSlicePreserveTypes preserves the type values from the field parameters when marshaling slices. This is an
// option since it deviates from stdlib.
func WithMarshalSlicePreserveTypes(value bool) MarshalOpt {
	return func(opts *marshalOpts) {
		opts.slicePreserveTypes = value
	}
}

// WithMarshalSliceAllowStrings allows slices of strings when marshaling slices. This is an option since it deviates
// from stdlib.
func WithMarshalSliceAllowStrings(value bool) MarshalOpt {
	return func(opts *marshalOpts) {
		opts.sliceAllowStrings = value
	}
}
