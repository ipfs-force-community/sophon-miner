package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/sophon-miner/api"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
)

var ExampleValues = map[reflect.Type]interface{}{
	reflect.TypeOf(auth.Permission("")): auth.Permission("write"),
	reflect.TypeOf(""):                  "string value",
	reflect.TypeOf(uint64(42)):          uint64(42),
	reflect.TypeOf(byte(7)):             byte(7),
	reflect.TypeOf([]byte{}):            []byte("byte array"),
}

func addExample(v interface{}) {
	ExampleValues[reflect.TypeOf(v)] = v
}

func init() {
	addr, err := address.NewIDAddress(1234)
	if err != nil {
		panic(err)
	}

	ExampleValues[reflect.TypeOf(addr)] = addr

	addExample(sharedTypes.APIVersion(1))
	addExample(abi.ChainEpoch(10101))
	addExample(int64(9))
	addExample(true)

	// miner specific
	//si := uint64(12)
	//addExample(&si)
	//addExample(abi.ActorID(1000))
}

func GetAPIType() (i interface{}, t reflect.Type, permStruct []reflect.Type) {
	i = &api.MinerAPIStruct{}
	t = reflect.TypeOf(new(struct{ api.MinerAPI })).Elem()
	permStruct = append(permStruct, reflect.TypeOf(api.MinerAPIStruct{}.Internal))
	permStruct = append(permStruct, reflect.TypeOf(api.CommonStruct{}.Internal))

	return
}

func ExampleValue(method string, t, parent reflect.Type) interface{} {
	v, ok := ExampleValues[t]
	if ok {
		return v
	}

	switch t.Kind() {
	case reflect.Slice:
		out := reflect.New(t).Elem()
		out = reflect.Append(out, reflect.ValueOf(ExampleValue(method, t.Elem(), t)))
		return out.Interface()
	case reflect.Chan:
		return ExampleValue(method, t.Elem(), nil)
	case reflect.Struct:
		es := exampleStruct(method, t, parent)
		v := reflect.ValueOf(es).Elem().Interface()
		ExampleValues[t] = v
		return v
	case reflect.Array:
		out := reflect.New(t).Elem()
		for i := 0; i < t.Len(); i++ {
			out.Index(i).Set(reflect.ValueOf(ExampleValue(method, t.Elem(), t)))
		}
		return out.Interface()

	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Struct {
			es := exampleStruct(method, t.Elem(), t)
			//ExampleValues[t] = es
			return es
		}
	case reflect.Interface:
		return struct{}{}
	}

	panic(fmt.Sprintf("No example value for type: %s (method '%s')", t, method))
}

func exampleStruct(method string, t, parent reflect.Type) interface{} {
	ns := reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if shouldIgnoreField(f, parent) {
			continue
		}

		caser := cases.Title(language.English)
		if caser.String(f.Name) == f.Name {
			ns.Elem().Field(i).Set(reflect.ValueOf(ExampleValue(method, f.Type, t)))
		}
	}

	return ns.Interface()
}

func shouldIgnoreField(f reflect.StructField, parentType reflect.Type) bool {
	if f.Type == parentType {
		return true
	}

	if len(f.Name) == 0 {
		return true
	}

	if !token.IsExported(f.Name) {
		return true
	}

	jtag := f.Tag.Get("json")
	if len(jtag) == 0 {
		return false
	}

	return strings.Split(jtag, ",")[0] == "-"
}

type Visitor struct {
	Root    string
	Methods map[string]ast.Node
}

func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	st, ok := node.(*ast.TypeSpec)
	if !ok {
		return v
	}

	if st.Name.Name != v.Root {
		return nil
	}

	iface := st.Type.(*ast.InterfaceType)
	for _, m := range iface.Methods.List {
		if len(m.Names) > 0 {
			v.Methods[m.Names[0].Name] = m
		}
	}

	return v
}

const NoComment = "There are not yet any comments for this method."

func ParseApiASTInfo() (comments map[string]string, groupDocs map[string]string) { //nolint:golint
	fset := token.NewFileSet()
	apiDir, err := filepath.Abs("api")
	if err != nil {
		fmt.Println("filepath absolute error: ", err)
		return
	}
	apiFile, err := filepath.Abs("api/api_miner.go")
	if err != nil {
		fmt.Println("filepath absolute error: ", err, "file:", apiFile)
		return
	}
	pkgs, err := parser.ParseDir(fset, apiDir, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		fmt.Println("parse error: ", err)
		return
	}

	ap := pkgs["api"]
	f := ap.Files[apiFile]
	cmap := ast.NewCommentMap(fset, f, f.Comments)

	v := &Visitor{"MinerAPI", make(map[string]ast.Node)}
	ast.Walk(v, ap)

	comments = make(map[string]string)
	groupDocs = make(map[string]string)
	for mn, node := range v.Methods {
		filteredComments := cmap.Filter(node).Comments()
		if len(filteredComments) == 0 {
			comments[mn] = NoComment
		} else {
			for _, c := range filteredComments {
				if strings.HasPrefix(c.Text(), "MethodGroup:") {
					parts := strings.Split(c.Text(), "\n")
					groupName := strings.TrimSpace(parts[0][12:])
					comment := strings.Join(parts[1:], "\n")
					groupDocs[groupName] = comment

					break
				}
			}

			l := len(filteredComments) - 1
			if len(filteredComments) > 1 {
				l = len(filteredComments) - 2
			}
			last := filteredComments[l].Text()
			if !strings.HasPrefix(last, "MethodGroup:") {
				comments[mn] = last
			} else {
				comments[mn] = NoComment
			}
		}
	}
	return comments, groupDocs
}

type MethodGroup struct {
	GroupName string
	Header    string
	Methods   []*Method
}

type Method struct {
	Comment         string
	Name            string
	InputExample    string
	ResponseExample string
}

func MethodGroupFromName(mn string) string {
	i := strings.IndexFunc(mn[1:], func(r rune) bool {
		return unicode.IsUpper(r)
	})
	if i < 0 {
		return ""
	}
	return mn[:i+1]
}
