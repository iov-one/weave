package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage:\n\t%s <pkgname>\n", os.Args[0])
		os.Exit(2)
	}
	pkgname := os.Args[1]

	ext, err := parseProtobuf(pkgname)
	if err != nil {
		panic(err)
	}

	files := []string{
		"doc.go",
		"init.go",
		"msg.go",
		"model.go",
		"handler.go",
		"configuration.go",
	}

	for _, name := range files {
		filepath := path.Join(pkgname, name)
		func() {
			fd, err := os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0555)
			if err != nil {
				panic(err)
			}
			defer fd.Close()

			if err := tmpl.ExecuteTemplate(fd, name, ext); err != nil {
				panic(err)
			}
		}()
	}
}

func parseProtobuf(pkgname string) (*Extension, error) {
	protopath := path.Join(pkgname, "codec.proto")
	ext := Extension{
		PackageName: pkgname,
		Messages:    []Message{},
		Models:      []Model{},
	}

	b, err := ioutil.ReadFile(protopath)
	if err != nil {
		return nil, fmt.Errorf("cannot read proto file: %w", err)
	}

	messages := regexp.MustCompile(`message (\S+) \{`).FindAllStringSubmatch(string(b), -1)
	for _, m := range messages {
		name := m[1]
		if strings.HasSuffix(name, "Msg") {
			hn := strings.ToLower(name[:1]) + name[1:len(name)-3] + "Handler"
			ext.Messages = append(ext.Messages, Message{
				PackageName: pkgname,
				Name:        name,
				HandlerName: hn,
			})
		} else {
			ext.Models = append(ext.Models, Model{
				PackageName: pkgname,
				Name:        name,
				BucketName:  strings.ToLower(name),
			})
		}
	}

	return &ext, nil
}

type Extension struct {
	PackageName string
	Messages    []Message
	Models      []Model
}

type Message struct {
	PackageName string
	Name        string
	HandlerName string
}

type Model struct {
	PackageName string
	Name        string
	BucketName  string
}

var tmpl = template.Must(template.New("").Parse(`
{{define "doc.go"}}
/*
Package {{.PackageName}} implements ...
*/
package {{.PackageName}}
{{end}}


{{define "msg.go"}}
package {{.PackageName}}

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
{{- range .Messages}}
	migration.MustRegister(1, &{{.Name}}{}, migration.NoModification)
{{- end}}
}


{{range .Messages}}
var _ weave.Msg = (*{{.Name}})(nil)

func (m *{{.Name}}) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", msg.Metadata.Validate())
	panic("TODO")
	return errs
}

{{end}}
{{end}}

{{define "model.go"}}
package {{.PackageName}}

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
{{- range .Models}}
	migration.MustRegister(1, &{{.Name}}{}, migration.NoModification)
{{- end}}
}

{{range .Models}}
var _ orm.Model = (*{{.Name}})(nil)

func (m *{{.Name}}) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	panic("TODO")
	return errs
}

func New{{.Name}}Bucket() orm.ModelBucket {
	b := orm.NewModelBucket("{{.BucketName}}", &{{.Name}}{})
	return migration.NewModelBucket("{{.PackageName}}", b)
}
{{end}}
{{end}}

{{define "handler.go"}}
func RegisterQuery(qr weave.QueryRouter) {
	{{- range .Models}}
	// New{{.Name}}Bucket().Register("{{.Name}}", qr)
	{{- end}}
}

func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry("{{.PackageName}}", r)
	{{- range .Messages}}
	r.Handle(&{{.HandlerName}}{
		auth:   auth,
	})
	{{- end}}
}



{{range .Messages}}
type {{.HandlerName}} struct {
	auth   x.Authenticator
}

func (h *{{.HandlerName}}) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	if _, err := h.validate(ctx, db, tx); err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: 0}, nil
}

func (h *{{.HandlerName}}) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	panic("TODO")
	return &weave.DeliverResult{Data: nil}, nil
}

func (h *{{.HandlerName}}) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*{{.Name}}, error) {
	var msg {{.Name}}
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}
	return &msg, nil
}
{{end}}
{{end}}

{{define "init.go"}}
package {{.PackageName}}

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

// Initializer fulfils the Initializer interface to load data from the genesis
// file
type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial account info from genesis and save it to the
// database
func (*Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, kv weave.KVStore) error {
	panic("not implemented")
}
{{end}}

{{define "configuration.go"}}
package {{.PackageName}}
{{end}}
`))
