package activities

import "fmt"

// errNotImplemented is the deliberate stub for units the generator did not
// synthesize (P3.8 review stop, MANUAL coverage, or an offline run without
// LLM bodies). It fails fast instead of silently doing nothing.
func errNotImplemented(what string) error {
	return fmt.Errorf("shinro: %s not implemented — manual migration step", what)
}
