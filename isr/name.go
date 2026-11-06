package isr

import "github.com/HazelnutParadise/insyra"

// NameSelector is insyra.NameSelector, so a selector built here works
// everywhere in the library and one built there works here.
type NameSelector = insyra.NameSelector

// Name says that a selector is a name rather than an index, exactly as
// insyra.Name does.
func Name(value string) NameSelector { return insyra.Name(value) }
