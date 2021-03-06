package ast

import (
	"github.com/velour/stop/token"
)

var (
	// UnivScope is the universal symtab, containing all predeclared identifiers.
	univScope = symtab{
		Decls: map[string]Declaration{
			// Predeclared types.
			"bool":       Bool,
			"byte":       Uint8,
			"complex64":  Complex64,
			"complex128": Complex128,
			"error":      Error,
			"float32":    Float32,
			"float64":    Float64,
			"int":        Int,
			"int8":       Int8,
			"int16":      Int16,
			"int32":      Int32,
			"int64":      Int64,
			"rune":       Int32,
			"string":     String,
			"uint":       Uint,
			"uint8":      Uint8,
			"uint16":     Uint16,
			"uint32":     Uint32,
			"uint64":     Uint64,
			"uintptr":    Uintptr,

			// Predeclared constants.
			"true":  &predeclaredConst{},
			"false": &predeclaredConst{},
			"iota":  &predeclaredConst{},

			// Predeclared zero value.
			"nil": &predeclaredConst{},

			// Predeclared functions.
			"append":  &predeclaredFunc{},
			"cap":     &predeclaredFunc{},
			"close":   &predeclaredFunc{},
			"complex": &predeclaredFunc{},
			"copy":    &predeclaredFunc{},
			"delete":  &predeclaredFunc{},
			"imag":    &predeclaredFunc{},
			"len":     &predeclaredFunc{},
			"make":    &predeclaredFunc{},
			"new":     &predeclaredFunc{},
			"panic":   &predeclaredFunc{},
			"print":   &predeclaredFunc{},
			"println": &predeclaredFunc{},
			"real":    &predeclaredFunc{},
			"recover": &predeclaredFunc{},
		},
	}
)

// A predeclaredType is a declaration node representing a predeclared type.
type predeclaredType int

// Predeclared types.
const (
	Bool predeclaredType = iota
	Complex64
	Complex128
	Error
	Float32
	Float64
	Int
	Int8
	Int16
	Int32
	Int64
	String
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
)

func (predeclaredType) Comments() []string    { return nil }
func (predeclaredType) Start() token.Location { return token.Location{} }
func (predeclaredType) End() token.Location   { return token.Location{} }

// A predeclaredConst is a declaration node representing a predeclared constant.
type predeclaredConst struct{}

func (*predeclaredConst) Comments() []string    { return nil }
func (*predeclaredConst) Start() token.Location { return token.Location{} }
func (*predeclaredConst) End() token.Location   { return token.Location{} }

// A predeclaredFunc is a declaration node representing a predeclared function.
type predeclaredFunc struct{}

func (*predeclaredFunc) Comments() []string    { return nil }
func (*predeclaredFunc) Start() token.Location { return token.Location{} }
func (*predeclaredFunc) End() token.Location   { return token.Location{} }

// A symtab is the main element of the symbol table. It contains a mapping
// from all identifiers in a given scope to their declaration. The declarations
// are unique.  Each identifier declared in a VarSpec or a ConstSpec is
// mapped to a unique view of the declaring spec.
type symtab struct {
	Up    *symtab
	Decls map[string]Declaration
}

// MakeSymtab returns a new symbol table.
func makeSymtab(up *symtab) *symtab {
	return &symtab{Up: up, Decls: make(map[string]Declaration)}
}

// Find returns the declaration bound to the given identifier, or nil if the identifier
// is not found.
func (s *symtab) Find(n string) Declaration {
	if s == nil {
		return nil
	}
	if d, ok := s.Decls[n]; ok {
		return d
	}
	return s.Up.Find(n)
}

// Bind binds a name to its declaration, returning an error if the name is already
// bound in this symtab. The blank identifier is not bound.
func (s *symtab) Bind(n string, decl Declaration) error {
	if n == "_" {
		return nil
	}
	if d, ok := s.Decls[n]; ok {
		return &Redeclaration{Name: n, First: d, Second: decl}
	}
	s.Decls[n] = decl
	return nil
}

// A checkState specifies whether a declaration is unchecked, checked,
// or is currently being checked.
type checkState int

const (
	unchecked = iota
	checking
	checkedOK
	checkedError
)

// A constSpecView is a view of a ConstSpec that focuses on a single identifier
// at a given index.
type constSpecView struct {
	*ConstSpec
	Index int
	// If the type of the const spec is not specified, then each identifier gets
	// its own type, based on the type of its expression.
	Type Type
	// Value is the value of the expression after constant folding. It is set
	// by the Check pass.
	Value Expression

	state checkState
}

// A varSpecView is a view of a VarSpec that focuses on a single identifier
// at a given index.
type varSpecView struct {
	Index int
	*VarSpec
}

// A packageDecl is a a package import declaration. It contains a mapping for all
// exported symbols in the package.
type packageDecl struct {
	syms *symtab
	*ImportDecl
}

// PkgDecls returns a symtab, mapping from package-scoped identifiers
// to their corresponding declarations. Each identifier is mapped to a
// unique declaration. Each identifier declared in a VarSpec or a
// ConstSpec is mapped to a unique view of the declaring spec.
// Any errors that are encountered are also returned, but the symtab is always
// valid, even in the face of errors.
func pkgDecls(files []*File) (*symtab, error) {
	psyms := makeSymtab(&univScope)
	var errs errors
	for _, f := range files {
		var err error
		if f.syms, err = fileDecls(psyms, f); err != nil {
			errs = append(errs, err)
		}
		for _, d := range f.Declarations {
			switch d := d.(type) {
			case *MethodDecl:
				d.syms = f.syms
				if err := psyms.Bind(d.Identifier.Name, d); err != nil {
					errs = append(errs, err)
				}
			case *FunctionDecl:
				d.syms = f.syms
				if err := psyms.Bind(d.Identifier.Name, d); err != nil {
					errs = append(errs, err)
				}
			case *TypeSpec:
				d.syms = f.syms
				if err := psyms.Bind(d.Identifier.Name, d); err != nil {
					errs = append(errs, err)
				}
			case *ConstSpec:
				d.syms = f.syms
				for i := range d.Identifiers {
					n := d.Identifiers[i].Name
					v := &constSpecView{Index: i, ConstSpec: d}
					d.views = append(d.views, v)
					if err := psyms.Bind(n, v); err != nil {
						errs = append(errs, err)
					}
				}
			case *VarSpec:
				d.syms = f.syms
				for i := range d.Identifiers {
					n := d.Identifiers[i].Name
					v := &varSpecView{Index: i, VarSpec: d}
					if err := psyms.Bind(n, v); err != nil {
						errs = append(errs, err)
					}
				}
			default:
				panic("invalid top-level declaration")
			}
		}
	}
	return psyms, errs.ErrorOrNil()
}

// FileDecls returns the symtab, mapping file-scoped identifiers to their
// correpsonding declarations. Any errors that are encountered are also
// returned, but the symtab is always valid, even in the face of errors.
func fileDecls(psyms *symtab, file *File) (*symtab, error) {
	syms := makeSymtab(psyms)
	var errs errors
	for i, d := range file.Imports {
		for _, im := range d.Imports {
			// BUG(eaburns): Should actually read the package imports.
			// BUG(eaburns): Should deal with dot imports
			p := &packageDecl{
				syms:       makeSymtab(&univScope),
				ImportDecl: &file.Imports[i],
			}
			if err := syms.Bind(im.Name(), p); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return syms, errs.ErrorOrNil()
}
