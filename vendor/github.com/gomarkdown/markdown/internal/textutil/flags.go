package textutil

// FlagName is the name of a bit flag.
type FlagName[F ~int] struct {
	Flag F
	Name string
}

// FlagNames returns, in table order, the names of the flags set in flags.
func FlagNames[F ~int](flags F, table []FlagName[F]) []string {
	var names []string
	for _, entry := range table {
		if flags&entry.Flag != 0 {
			names = append(names, entry.Name)
		}
	}
	return names
}
