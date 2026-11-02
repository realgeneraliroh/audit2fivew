// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

package avc

import (
	"fmt"
	"strings"
)

// GenerateModuleName returns a sanitised module name for the TE rule
// derived from the source and target SELinux types.
func (d *Denial) GenerateModuleName() string {
	stype := sanitize(d.SType())
	ttype := sanitize(d.TType())
	return fmt.Sprintf("my_%s_%s_fix", stype, ttype)
}

// GenerateTE produces a complete Type Enforcement (.te) policy module
// that would allow the denied action.  This replicates the output of
// audit2allow for a single denial.
func (d *Denial) GenerateTE() string {
	modName := d.GenerateModuleName()
	stype := d.SType()
	ttype := d.TType()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("module %s 1.0;\n\n", modName))
	b.WriteString("require {\n")
	b.WriteString(fmt.Sprintf("\ttype %s;\n", stype))
	if stype != ttype {
		b.WriteString(fmt.Sprintf("\ttype %s;\n", ttype))
	}
	b.WriteString(fmt.Sprintf("\tclass %s { %s };\n", d.TClass, d.PermissionsStr()))
	b.WriteString("}\n\n")
	b.WriteString(fmt.Sprintf("#============= %s ==============\n", stype))
	b.WriteString(fmt.Sprintf("allow %s %s:%s { %s };\n",
		stype, ttype, d.TClass, d.PermissionsStr()))

	return b.String()
}

// GenerateCommands returns the shell commands needed to compile and
// install the generated TE module via checkmodule / semodule_package /
// semodule.
func (d *Denial) GenerateCommands() string {
	mod := d.GenerateModuleName()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# 1. Save the .te rule above to %s.te, then run:\n", mod))
	b.WriteString(fmt.Sprintf("checkmodule -M -m -o %s.mod %s.te\n", mod, mod))
	b.WriteString(fmt.Sprintf("semodule_package -o %s.pp -m %s.mod\n", mod, mod))
	b.WriteString(fmt.Sprintf("semodule -i %s.pp\n", mod))
	return b.String()
}

// sanitize replaces any character that is not alphanumeric or underscore
// with an underscore, producing a string safe for use as a TE module name.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, s)
}

