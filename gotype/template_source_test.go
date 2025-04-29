package gotype

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestGetterFunc(t *testing.T) {
	t.Run("Basic cases", func(t *testing.T) {
		type args struct {
			name string
			t    types.Type
		}
		tests := []struct {
			name string
			args args
			want string
		}{
			{
				name: "Returns empty string for non-struct type",
				args: args{
					name: "MyType",
					t:    types.Typ[types.String],
				},
				want: "",
			},
			{
				name: "Struct with basic type fields",
				args: args{
					name: "Person",
					t: types.NewStruct([]*types.Var{
						types.NewField(0, nil, "Name", types.Typ[types.String], false),
					}, nil),
				},
				want: "func (t *Person) GetName() string {\n\tif t == nil {\n\t\tt = &Person{}\n\t}\n\treturn t.Name\n}\n",
			},
			{
				name: "Struct with named type fields",
				args: args{
					name: "User",
					t: types.NewStruct([]*types.Var{
						types.NewField(0, nil, "Profile", types.NewNamed(types.NewTypeName(0, nil, "Profile", nil), nil, nil), false),
					}, nil),
				},
				want: "func (t *User) GetProfile() *Profile {\n\tif t == nil {\n\t\tt = &User{}\n\t}\n\treturn &t.Profile\n}\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				getterFunc := GetterFunc()
				if got := getterFunc(tt.args.name, tt.args.t); got != tt.want {
					t.Errorf("GetterFunc() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Real struct cases", func(t *testing.T) {
		// Get Types information by parsing package from source code
		fset := token.NewFileSet()

		// Source code containing test struct definitions
		src := `
package testpkg

type SimpleUser struct {
	ID   string
	Name string
	Age  int
}

type ComplexUser struct {
	ID      string
	Profile Profile
	Active  bool
}

type Profile struct {
	Bio  string
	Tags []string
}
`

		f, err := parser.ParseFile(fset, "testpkg.go", src, 0)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		conf := types.Config{Importer: importer.Default()}
		pkg, err := conf.Check("testpkg", fset, []*ast.File{f}, nil)
		if err != nil {
			t.Fatalf("failed to check package: %v", err)
		}

		// Get type information for SimpleUser struct
		simpleUserObj := pkg.Scope().Lookup("SimpleUser")
		if simpleUserObj == nil {
			t.Fatalf("SimpleUser not found")
		}
		simpleUserType := simpleUserObj.Type().Underlying().(*types.Struct)

		// Get type information for ComplexUser struct
		complexUserObj := pkg.Scope().Lookup("ComplexUser")
		if complexUserObj == nil {
			t.Fatalf("ComplexUser not found")
		}
		complexUserType := complexUserObj.Type().Underlying().(*types.Struct)

		tests := []struct {
			name       string
			structName string
			t          *types.Struct
			want       string
		}{
			{
				name:       "SimpleUser with basic type fields",
				structName: "SimpleUser",
				t:          simpleUserType,
				want: `func (t *SimpleUser) GetID() string {
	if t == nil {
		t = &SimpleUser{}
	}
	return t.ID
}
func (t *SimpleUser) GetName() string {
	if t == nil {
		t = &SimpleUser{}
	}
	return t.Name
}
func (t *SimpleUser) GetAge() int {
	if t == nil {
		t = &SimpleUser{}
	}
	return t.Age
}
`,
			},
			{
				name:       "ComplexUser with nested struct",
				structName: "ComplexUser",
				t:          complexUserType,
				want: `func (t *ComplexUser) GetID() string {
	if t == nil {
		t = &ComplexUser{}
	}
	return t.ID
}
func (t *ComplexUser) GetProfile() *testpkg.Profile {
	if t == nil {
		t = &ComplexUser{}
	}
	return &t.Profile
}
func (t *ComplexUser) GetActive() bool {
	if t == nil {
		t = &ComplexUser{}
	}
	return t.Active
}
`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				getterFunc := GetterFunc()
				got := getterFunc(tt.structName, tt.t)
				if got != tt.want {
					t.Errorf("GetterFunc() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Types in the same package", func(t *testing.T) {
		fset := token.NewFileSet()

		// Source code with all types in the same package
		src := `
package samepkg

type User struct {
	ID      string
	Address Address
}

type Address struct {
	Street  string
	City    string
	Country string
}
`

		f, err := parser.ParseFile(fset, "samepkg.go", src, 0)
		if err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}

		conf := types.Config{Importer: importer.Default()}
		pkg, err := conf.Check("samepkg", fset, []*ast.File{f}, nil)
		if err != nil {
			t.Fatalf("failed to check package: %v", err)
		}

		// Get type information for User struct
		userObj := pkg.Scope().Lookup("User")
		if userObj == nil {
			t.Fatalf("User not found")
		}
		userType := userObj.Type().Underlying().(*types.Struct)

		// Run test
		getterFunc := GetterFunc()
		got := getterFunc("User", userType)

		// Expected result: Address without package qualifier
		want := `func (t *User) GetID() string {
	if t == nil {
		t = &User{}
	}
	return t.ID
}
func (t *User) GetAddress() *Address {
	if t == nil {
		t = &User{}
	}
	return &t.Address
}
`

		if got != want {
			t.Errorf("GetterFunc() = %v, want %v", got, want)
		}

		// Verify that namedTypeString function properly handles package qualifiers
		// No package qualifier for types referenced within the package
		addressObj := pkg.Scope().Lookup("Address")
		if addressObj == nil {
			t.Fatalf("Address not found")
		}

		addressNamed := addressObj.Type().(*types.Named)
		// Specify the same package path
		result := ref(addressNamed)

		// For same package, should return type name only without package qualifier
		if result != "Address" {
			t.Errorf("namedTypeString() for same package = %v, want %v", result, "Address")
		}

		// For different package, should include package qualifier
		resultWithPkg := ref(addressNamed)
		if resultWithPkg != "samepkg.Address" {
			t.Errorf("namedTypeString() for different package = %v, want %v", resultWithPkg, "samepkg.Address")
		}
	})

	t.Run("Types in different packages", func(t *testing.T) {
		// First check if namedTypeString function works correctly
		pkgB := types.NewPackage("github.com/example/pkgb", "pkgb")
		profileTypeName := types.NewTypeName(0, pkgB, "Profile", nil)
		profileType := types.NewNamed(profileTypeName, nil, nil)

		// For same package reference, no package qualifier
		samePkg := ref(profileType)
		if samePkg != "Profile" {
			t.Errorf("namedTypeString() for same package = %v, want %v", samePkg, "Profile")
		}

		// For different package reference, include package qualifier
		differentPkg := ref(profileType)
		if differentPkg != "pkgb.Profile" {
			t.Errorf("namedTypeString() for different package = %v, want %v", differentPkg, "pkgb.Profile")
		}

		// Next check if funcReturnTypesName function works correctly
		// No package specified
		noPackage := ref(profileType)
		if noPackage != "*pkgb.Profile" {
			t.Errorf("funcReturnTypesName() without pkg path = %v, want %v", noPackage, "*pkgb.Profile")
		}

		// Same package
		samePackageType := ref(profileType)
		if samePackageType != "*Profile" {
			t.Errorf("funcReturnTypesName() with same pkg = %v, want %v", samePackageType, "*Profile")
		}

		// Different package
		differentPackageType := ref(profileType)
		if differentPackageType != "*pkgb.Profile" {
			t.Errorf("funcReturnTypesName() with different pkg = %v, want %v", differentPackageType, "*pkgb.Profile")
		}

		// Finally test GetterFunc function
		// Create user struct
		userStructFields := []*types.Var{
			types.NewField(0, nil, "ID", types.Typ[types.String], false),
			types.NewField(0, nil, "Profile", profileType, false),
		}
		userStruct := types.NewStruct(userStructFields, nil)

		// Call GetterFunc with same package
		getterFuncSamePkg := GetterFunc()
		gotWithSamePkg := getterFuncSamePkg("User", userStruct)

		// Expected value when package qualifier is omitted for same package
		wantWithSamePkg := `func (t *User) GetID() string {
	if t == nil {
		t = &User{}
	}
	return t.ID
}
func (t *User) GetProfile() *Profile {
	if t == nil {
		t = &User{}
	}
	return &t.Profile
}
`
		if gotWithSamePkg != wantWithSamePkg {
			t.Errorf("GetterFunc() with same package = %v, want %v", gotWithSamePkg, wantWithSamePkg)
		}

		// Call GetterFunc with different package
		getterFuncDiffPkg := GetterFunc()
		gotWithDifferentPkg := getterFuncDiffPkg("User", userStruct)

		// Expected value when package qualifier is included for different package
		wantWithDifferentPkg := `func (t *User) GetID() string {
	if t == nil {
		t = &User{}
	}
	return t.ID
}
func (t *User) GetProfile() *pkgb.Profile {
	if t == nil {
		t = &User{}
	}
	return &t.Profile
}
`
		if gotWithDifferentPkg != wantWithDifferentPkg {
			t.Errorf("GetterFunc() with different package = %v, want %v", gotWithDifferentPkg, wantWithDifferentPkg)
		}
	})
}
