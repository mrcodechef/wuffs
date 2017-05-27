// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

//go:generate go run gen.go

package cgen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	a "github.com/google/puffs/lang/ast"
	t "github.com/google/puffs/lang/token"
)

type Generator struct {
	// Extension should be either 'c' or 'h'.
	Extension byte
}

func (g *Generator) Generate(pkgName string, m *t.IDMap, files []*a.File) ([]byte, error) {
	b := &bytes.Buffer{}

	// Write preamble.
	fmt.Fprintf(b, "// Code generated by puffs-gen-%c. DO NOT EDIT.\n\n", g.Extension)
	b.WriteString(preamble)
	b.WriteString("\n#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")

	b.WriteString("// ---------------- Status Codes\n\n")
	b.WriteString("// Status codes are non-positive integers.\n")
	b.WriteString("//\n")
	b.WriteString("// The least significant bit indicates a fatal status code: an error.\n")
	b.WriteString("typedef enum {\n")
	fmt.Fprintf(b, "puffs_%s_status_ok = 0,\n", pkgName)
	fmt.Fprintf(b, "puffs_%s_error_bad_version = -1,\n", pkgName)
	fmt.Fprintf(b, "puffs_%s_status_short_dst = -4,\n", pkgName)
	fmt.Fprintf(b, "puffs_%s_status_short_src = -6,\n", pkgName)
	fmt.Fprintf(b, "} puffs_%s_status;\n\n", pkgName)

	b.WriteString("// ---------------- Structs\n\n")
	if err := forEachStruct(b, pkgName, m, files, writeStruct); err != nil {
		return nil, err
	}

	b.WriteString("// ---------------- Constructor and Destructor Prototypes\n\n")
	if err := forEachStruct(b, pkgName, m, files, writeCtorPrototypes); err != nil {
		return nil, err
	}

	// Finish up the header, which is also the first part of the .c file.
	b.WriteString("\n#ifdef __cplusplus\n}  // extern \"C\"\n#endif\n\n")
	if g.Extension == 'h' {
		return format(b)
	}

	b.WriteString("// ---------------- Constructor and Destructor Implementations\n\n")
	b.WriteString("// PUFFS_MAGIC is a magic number to check that constructors are called. It's\n")
	b.WriteString("// not foolproof, given C doesn't automatically zero memory before use, but it\n")
	b.WriteString("// should catch 99.99% of cases.\n")
	b.WriteString("//\n")
	b.WriteString("// Its (non-zero) value is arbitrary, based on md5sum(\"puffs\").\n")
	b.WriteString("#define PUFFS_MAGIC (0xCB3699CCU)\n\n")
	if err := forEachStruct(b, pkgName, m, files, writeCtorImpls); err != nil {
		return nil, err
	}

	return format(b)
}

func forEachStruct(b *bytes.Buffer, pkgName string, m *t.IDMap, files []*a.File,
	f func(*bytes.Buffer, string, *t.IDMap, *a.Struct) error) error {

	for _, file := range files {
		for _, n := range file.TopLevelDecls() {
			if n.Kind() == a.KStruct {
				if err := f(b, pkgName, m, n.Struct()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeStruct(b *bytes.Buffer, pkgName string, m *t.IDMap, n *a.Struct) error {
	structName := n.Name().String(m)
	fmt.Fprintf(b, "typedef struct {\n")
	if n.Suspendible() {
		fmt.Fprintf(b, "puffs_%s_status status;\n", pkgName)
		fmt.Fprintf(b, "uint32_t magic;\n")
	}
	for _, f := range n.Fields() {
		if err := writeField(b, m, f.Field()); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "} puffs_%s_%s;\n\n", pkgName, structName)
	return nil
}

func writeCtorSignature(b *bytes.Buffer, pkgName string, m *t.IDMap, n *a.Struct, ctor string) {
	structName := n.Name().String(m)
	fmt.Fprintf(b, "void puffs_%s_%s_%s(puffs_%s_%s *self", pkgName, structName, ctor, pkgName, structName)
	if ctor == "constructor" {
		fmt.Fprintf(b, ", uint32_t puffs_version")
	}
	fmt.Fprintf(b, ")")
}

func writeCtorPrototypes(b *bytes.Buffer, pkgName string, m *t.IDMap, n *a.Struct) error {
	if !n.Suspendible() {
		return nil
	}
	for _, ctor := range []string{"constructor", "destructor"} {
		writeCtorSignature(b, pkgName, m, n, ctor)
		b.WriteString(";\n\n")
	}
	return nil
}

func writeCtorImpls(b *bytes.Buffer, pkgName string, m *t.IDMap, n *a.Struct) error {
	if !n.Suspendible() {
		return nil
	}
	for _, ctor := range []string{"constructor", "destructor"} {
		writeCtorSignature(b, pkgName, m, n, ctor)
		fmt.Fprintf(b, "{\n")

		if ctor == "constructor" {
			fmt.Fprintf(b, "if (puffs_version != PUFFS_VERSION) {\n")
			fmt.Fprintf(b, "self->status = puffs_%s_error_bad_version;\n", pkgName)
			fmt.Fprintf(b, "return;\n")
			fmt.Fprintf(b, "}\n")

			b.WriteString("memset(self, 0, sizeof(*self));\n")
			b.WriteString("self->magic = PUFFS_MAGIC;\n")

			// TODO: set any non-zero default values.
		}

		// TODO: call any ctor/dtors on sub-structures.
		b.WriteString("}\n\n")
	}
	return nil
}

func writeField(b *bytes.Buffer, m *t.IDMap, n *a.Field) error {
	convertible := true
	for x := n.XType(); x != nil; x = x.Inner() {
		if p := x.PackageOrDecorator(); p != 0 && p != t.IDOpenBracket {
			convertible = false
			break
		}
		if x.Inner() != nil {
			continue
		}
		if k := x.Name().Key(); k < t.Key(len(cTypeNames)) {
			if s := cTypeNames[k]; s != "" {
				b.WriteString(s)
				b.WriteByte(' ')
				continue
			}
		}
		convertible = false
		break
	}
	if !convertible {
		// TODO: fix this.
		return fmt.Errorf("cannot convert Puffs type %q to C", n.XType().String(m))
	}

	b.WriteString("f_")
	b.WriteString(n.Name().String(m))

	for x := n.XType(); x != nil; x = x.Inner() {
		if x.PackageOrDecorator() == t.IDOpenBracket {
			b.WriteByte('[')
			b.WriteString(x.ArrayLength().ConstValue().String())
			b.WriteByte(']')
		}
	}

	b.WriteString(";\n")
	return nil
}

func format(rawSource *bytes.Buffer) ([]byte, error) {
	stdout := &bytes.Buffer{}
	cmd := exec.Command("clang-format", "-style=Chromium")
	cmd.Stdin = rawSource
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

var cTypeNames = [...]string{
	t.KeyI8:    "int8_t",
	t.KeyI16:   "int16_t",
	t.KeyI32:   "int32_t",
	t.KeyI64:   "int64_t",
	t.KeyU8:    "uint8_t",
	t.KeyU16:   "uint16_t",
	t.KeyU32:   "uint32_t",
	t.KeyU64:   "uint64_t",
	t.KeyUsize: "size_t",
	t.KeyBool:  "bool",
}
