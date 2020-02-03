package entity

import (
	"fmt"
	"strings"
	"text/template"

	proto "github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
	"github.com/pkg/errors"
	"github.com/utilitywarehouse/protoc-gen-uwgo-entity/protos"
)

const module = "uw.entity_identifier"

var fieldTypes = map[pgs.ProtoType]struct {
	returnType    string
	defaultReturn string
}{
	pgs.StringT: {
		returnType:    "string",
		defaultReturn: `""`,
	},
}

// IdentifierModule validates & generates code for accessing an identifier on a message
type IdentifierModule struct {
	*pgs.ModuleBase
	ctx pgsgo.Context
	tpl *template.Template

	// options
	enforce       bool   // All messages must have an entity identifier set
	enforceSuffix string // If set, only messages with this suffix will be enforced
}

type entity struct {
	Msg    pgs.Message
	Entity pgs.Field
}

// NewIdentifierModule creates a module for PG*
func NewIdentifierModule() *IdentifierModule {
	return &IdentifierModule{
		ModuleBase: &pgs.ModuleBase{},
	}
}

// Name identifies this module
func (m *IdentifierModule) Name() string {
	return module
}

// InitContext sets up module for use
func (m *IdentifierModule) InitContext(c pgs.BuildContext) {
	m.ModuleBase.InitContext(c)
	m.ctx = pgsgo.InitContext(c.Parameters())

	// Options
	enforce, err := m.ctx.Params().Bool("enforce")
	if err != nil {
		m.AddError(err.Error())
		return
	}
	m.enforce = enforce
	m.enforceSuffix = m.ctx.Params().Str("enforce-suffix")
	if m.enforceSuffix != "" {
		m.enforce = true
	}

	tpl := template.New(module).Funcs(map[string]interface{}{
		"package":             m.ctx.PackageName,
		"entityReturnType":    entityReturnType,
		"entityDefaultReturn": entityDefaultReturn,
	})
	m.tpl = template.Must(tpl.Parse(entityTpl))
}

// Execute runs the generator
func (m *IdentifierModule) Execute(targets map[string]pgs.File, pkgs map[string]pgs.Package) []pgs.Artifact {
	for _, t := range targets {
		m.Debugf("generating for target: %s", t.Name())

		entities, err := m.generate(t)
		if err != nil {
			m.AddError(err.Error())
			break
		}

		m.render(t, entities)
	}

	return m.Artifacts()
}

func (m *IdentifierModule) generate(f pgs.File) ([]entity, error) {
	if len(f.Messages()) == 0 {
		m.Debugf("zero messages, skipping: %s", f.Name())
		return nil, nil
	}

	entities := make([]entity, 0, len(f.Messages()))

	for _, msg := range f.Messages() {
		// Ignore msg with `uw.entity.ignore = true`
		ignoreMsg, err := shouldIgnoreMsg(msg.Descriptor())
		if err != nil {
			return entities, err
		}
		if ignoreMsg {
			m.Debugf(fmt.Sprintf("%s:%s ignoring entity", f.Name(), msg.Name()))
			continue
		}

		if m.enforceSuffix != "" && !strings.HasSuffix(msg.Name().String(), m.enforceSuffix) {
			// suffix enforced, msg does not match
			continue
		}

		// Check each field on the message, this will be false
		// if we fail to find a valid option
		var hasEntityIdentifier bool

		for _, field := range msg.Fields() {
			fdesc := field.Descriptor()
			if fdesc == nil {
				continue
			}
			if fdesc.GetOptions() == nil {
				continue
			}

			ext, err := proto.GetExtension(fdesc.GetOptions(), protos.E_Identifier)
			if err != nil {
				if errors.Is(err, proto.ErrMissingExtension) {
					continue
				}
				return entities, err
			}
			if ext == nil {
				continue
			}

			isEntity, ok := ext.(*bool)
			if !ok || isEntity == nil {
				return entities, errors.New("invalid option type")
			}

			hasEntityIdentifier = *isEntity

			if hasEntityIdentifier {
				if !validFieldType(field) {
					return entities, errors.Errorf("unable to handle identifier field type: %s", field.Type().ProtoType())
				}

				entities = append(entities, entity{
					Msg:    msg,
					Entity: field,
				})
			}
		}

		if m.enforce && !hasEntityIdentifier {
			missingMsg := fmt.Sprintf("%s:%s `uw.entity.identifier` not set", f.Name(), msg.Name())
			if m.enforce {
				return entities, errors.New(missingMsg)
			}
			m.Debugf(missingMsg)
		}
	}

	return entities, nil
}

func (m *IdentifierModule) render(f pgs.File, entities []entity) {
	if len(entities) == 0 {
		return
	}

	name := m.ctx.OutputPath(f).SetExt(".uw.entity.go")
	m.AddGeneratorTemplateFile(name.String(), m.tpl, map[string]interface{}{
		"File":     f,
		"Entities": entities,
	})
}

func shouldIgnoreMsg(desc *descriptor.DescriptorProto) (bool, error) {
	if desc == nil {
		return false, nil
	}
	if desc.GetOptions() == nil {
		return false, nil
	}

	ext, err := proto.GetExtension(desc.GetOptions(), protos.E_Ignore)
	if err != nil {
		if errors.Is(err, proto.ErrMissingExtension) {
			return false, nil
		}
		return false, err
	}
	if ext == nil {
		return false, nil
	}

	ignore, ok := ext.(*bool)
	if !ok {
		return false, errors.New("invalid option type")
	}
	return *ignore, nil
}

func entityReturnType(field pgs.Field) string {
	fieldType, ok := fieldTypes[field.Type().ProtoType()]
	if !ok {
		return ""
	}
	return fieldType.returnType
}

func entityDefaultReturn(field pgs.Field) string {
	fieldType, ok := fieldTypes[field.Type().ProtoType()]
	if !ok {
		return ""
	}
	return fieldType.defaultReturn
}

func validFieldType(field pgs.Field) bool {
	_, ok := fieldTypes[field.Type().ProtoType()]
	return ok
}
