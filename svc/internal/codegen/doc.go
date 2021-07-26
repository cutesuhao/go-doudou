package codegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/sirupsen/logrus"
	"github.com/unionj-cloud/go-doudou/astutils"
	"github.com/unionj-cloud/go-doudou/constants"
	v3 "github.com/unionj-cloud/go-doudou/openapi/v3"
	"github.com/unionj-cloud/go-doudou/stringutils"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"time"
)

func getSchemaNames(vofile string) []string {
	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, vofile, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	sc := astutils.NewStructCollector(ExprStringP)
	ast.Walk(sc, root)
	structs := sc.DocFlatEmbed()
	var ret []string
	for _, item := range structs {
		ret = append(ret, item.Name)
	}
	return ret
}

func schemasOf(vofile string) []v3.Schema {
	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, vofile, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	sc := astutils.NewStructCollector(ExprStringP)
	ast.Walk(sc, root)
	structs := sc.DocFlatEmbed()
	var ret []v3.Schema
	for _, item := range structs {
		ret = append(ret, v3.NewSchema(item))
	}
	return ret
}

const (
	get    = "GET"
	post   = "POST"
	put    = "PUT"
	delete = "DELETE"
)

func operationOf(method astutils.MethodMeta, httpMethod string) v3.Operation {
	var ret v3.Operation
	var params []v3.Parameter

	ret.Summary = strings.Join(method.Comments, "\n")

	// If http method is "POST" and each parameters' type is one of v3.Int, v3.Int64, v3.Bool, v3.String, v3.Float32, v3.Float64,
	// then we use application/x-www-form-urlencoded as Content-type and we make one ref schema from them as request body.
	// Note: unionj-generator project hasn't support application/x-www-form-urlencoded yet
	var simpleCnt int
	for _, item := range method.Params {
		if v3.IsBuiltin(item) || item.Type == "context.Context" {
			simpleCnt++
		}
	}
	if httpMethod == post && simpleCnt == len(method.Params) {
		title := method.Name + "Req"
		reqSchema := v3.Schema{
			Type:       v3.ObjectT,
			Title:      title,
			Properties: make(map[string]*v3.Schema),
		}
		for _, item := range method.Params {
			if item.Type == "context.Context" {
				continue
			}
			key := item.Name
			pschema := v3.CopySchema(item)
			pschema.Description = strings.Join(item.Comments, "\n")
			reqSchema.Properties[strcase.ToLowerCamel(key)] = &pschema
		}
		v3.Schemas[title] = reqSchema
		mt := &v3.MediaType{
			Schema: &v3.Schema{
				Ref: "#/components/schemas/" + title,
			},
		}
		var content v3.Content
		reflect.ValueOf(&content).Elem().FieldByName("FormUrl").Set(reflect.ValueOf(mt))
		ret.RequestBody = &v3.RequestBody{
			Content:  &content,
			Required: true,
		}
	} else {
		// Simple parameters such as v3.Int, v3.Int64, v3.Bool, v3.String, v3.Float32, v3.Float64 and corresponding Array type
		// will be put into query parameter as url search params no matter what http method is.
		// Complex parameters such as structs in vo package, map and corresponding slice/array type
		// will be put into request body as json content type.
		// File and file array parameter will be put into request body as multipart/form-data content type.
		for _, item := range method.Params {
			if item.Type == "context.Context" {
				continue
			}
			pschemaType := v3.SchemaOf(item)
			pschema := v3.CopySchema(item)
			pschema.Description = strings.Join(item.Comments, "\n")
			if reflect.DeepEqual(pschemaType, v3.FileArray) || pschemaType == v3.File {
				var content v3.Content
				mt := &v3.MediaType{
					Schema: &pschema,
				}
				reflect.ValueOf(&content).Elem().FieldByName("FormData").Set(reflect.ValueOf(mt))
				ret.RequestBody = &v3.RequestBody{
					Content:  &content,
					Required: true,
				}
			} else if v3.IsBuiltin(item) {
				params = append(params, v3.Parameter{
					Name:   strcase.ToLowerCamel(item.Name),
					In:     v3.InQuery,
					Schema: &pschema,
				})
			} else {
				var content v3.Content
				mt := &v3.MediaType{
					Schema: &pschema,
				}
				reflect.ValueOf(&content).Elem().FieldByName("Json").Set(reflect.ValueOf(mt))
				ret.RequestBody = &v3.RequestBody{
					Content:  &content,
					Required: true,
				}
			}
		}
	}

	ret.Parameters = params
	var respContent v3.Content
	var hasFile bool
	var fileDoc string
	for _, item := range method.Results {
		if item.Type == "*os.File" {
			hasFile = true
			fileDoc = strings.Join(item.Comments, "\n")
			break
		}
	}
	if hasFile {
		respContent.Stream = &v3.MediaType{
			Schema: &v3.Schema{
				Type:        v3.StringT,
				Format:      v3.BinaryF,
				Description: fileDoc,
			},
		}
	} else {
		title := method.Name + "Resp"
		respSchema := v3.Schema{
			Type:       v3.ObjectT,
			Title:      title,
			Properties: make(map[string]*v3.Schema),
		}
		for _, item := range method.Results {
			key := item.Name
			if stringutils.IsEmpty(key) {
				key = item.Type[strings.LastIndex(item.Type, ".")+1:]
			}
			rschema := v3.CopySchema(item)
			rschema.Description = strings.Join(item.Comments, "\n")
			respSchema.Properties[strcase.ToLowerCamel(key)] = &rschema
		}
		v3.Schemas[title] = respSchema
		respContent.Json = &v3.MediaType{
			Schema: &v3.Schema{
				Ref: "#/components/schemas/" + title,
			},
		}
	}
	ret.Responses = &v3.Responses{
		Resp200: &v3.Response{
			Content: &respContent,
		},
	}
	return ret
}

func pathOf(method astutils.MethodMeta) v3.Path {
	var ret v3.Path
	hm := httpMethod(method.Name)
	op := operationOf(method, hm)
	reflect.ValueOf(&ret).Elem().FieldByName(strings.Title(strings.ToLower(hm))).Set(reflect.ValueOf(&op))
	return ret
}

func pathsOf(ic astutils.InterfaceCollector) map[string]v3.Path {
	if len(ic.Interfaces) == 0 {
		return nil
	}
	pathmap := make(map[string]v3.Path)
	inter := ic.Interfaces[0]
	for _, method := range inter.Methods {
		v3path := pathOf(method)
		endpoint := fmt.Sprintf("/%s/%s", strings.ToLower(inter.Name), pattern(method.Name))
		pathmap[endpoint] = v3path
	}
	return pathmap
}

var gofileTmpl = `package {{.SvcPackage}}

import "github.com/unionj-cloud/go-doudou/svc/http/onlinedoc"

func init() {
	onlinedoc.Oas = ` + "`" + `{{.Doc}}` + "`" + `
}
`

// Currently not suport alias type in vo file. TODO
func GenDoc(dir string, ic astutils.InterfaceCollector) {
	var (
		err     error
		svcname string
		docfile string
		gofile  string
		fi      os.FileInfo
		api     v3.Api
		data    []byte
		vos     []v3.Schema
		paths   map[string]v3.Path
		tpl     *template.Template
		sqlBuf  bytes.Buffer
		source  string
	)
	v3.Schemas = make(map[string]v3.Schema)
	svcname = ic.Interfaces[0].Name
	docfile = filepath.Join(dir, strings.ToLower(svcname)+"_openapi3.json")
	fi, err = os.Stat(docfile)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	if fi != nil {
		logrus.Warningln("file " + docfile + " will be overwrited")
	}
	gofile = filepath.Join(dir, strings.ToLower(svcname)+"_openapi3.go")
	fi, err = os.Stat(gofile)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	if fi != nil {
		logrus.Warningln("file " + gofile + " will be overwrited")
	}
	vodir := filepath.Join(dir, "vo")
	var files []string
	err = filepath.Walk(vodir, astutils.Visit(&files))
	if err != nil {
		logrus.Panicln(err)
	}
	for _, file := range files {
		v3.SchemaNames = append(v3.SchemaNames, getSchemaNames(file)...)
	}
	for _, file := range files {
		vos = append(vos, schemasOf(file)...)
	}
	for _, item := range vos {
		v3.Schemas[item.Title] = item
	}
	paths = pathsOf(ic)
	api = v3.Api{
		Openapi: "3.0.2",
		Info: &v3.Info{
			Title:          svcname,
			Description:    "",
			TermsOfService: "",
			Contact:        nil,
			License:        nil,
			Version:        fmt.Sprintf("v%s", time.Now().Local().Format(constants.FORMAT10)),
		},
		Paths: paths,
		Components: &v3.Components{
			Schemas: v3.Schemas,
		},
	}
	data, err = json.Marshal(api)
	err = ioutil.WriteFile(docfile, data, os.ModePerm)
	if err != nil {
		panic(err)
	}

	if tpl, err = template.New("doc.go.tmpl").Parse(gofileTmpl); err != nil {
		panic(err)
	}
	if err = tpl.Execute(&sqlBuf, struct {
		SvcPackage string
		Doc        string
	}{
		SvcPackage: ic.Package.Name,
		Doc:        string(data),
	}); err != nil {
		panic(err)
	}
	source = strings.TrimSpace(sqlBuf.String())
	astutils.FixImport([]byte(source), gofile)
}
