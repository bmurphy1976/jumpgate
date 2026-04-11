package handlers

import (
	"dashboard/internal/buildinfo"
	"dashboard/model"
	"fmt"
	"reflect"
	"strings"

	"github.com/labstack/echo/v5"
)

// API input/output types — used for request body binding and OpenAPI schema generation.

type CategoryCreateInput struct {
	Name string `json:"name" api:"required,description=Category name"`
}

type CategoryUpdateInput struct {
	Name *string `json:"name" api:"description=New category name"`
}

type BookmarkCreateInput struct {
	CategoryID int       `json:"category_id" api:"required,description=Category to add bookmark to"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	MobileURL  string    `json:"mobile_url"`
	Icon       string    `json:"icon" api:"description=MDI icon name"`
	Keywords   *[]string `json:"keywords"`
}

type BookmarkUpdateInput struct {
	Name      *string   `json:"name"`
	URL       *string   `json:"url"`
	MobileURL *string   `json:"mobile_url"`
	Icon      *string   `json:"icon" api:"description=MDI icon name"`
	Keywords  *[]string `json:"keywords"`
}

type BookmarkMoveInput struct {
	CategoryID int `json:"category_id" api:"required,description=Target category ID"`
	Position   int `json:"position"    api:"description=Position in target category (0 = end)"`
}

type IconListOutput struct {
	Icons []string `json:"icons"`
	Total int      `json:"total"`
}

// apiRoute pairs a route registration with its OpenAPI spec metadata.
type apiRoute struct {
	Method  string
	Path    string // Echo-style path without /api prefix
	Handler echo.HandlerFunc
	Write   bool
	Summary string
	Tag     string
	Status  int // HTTP success status code
	Input   any // nil, or struct instance for request body schema
	Output  any // nil, or struct instance for response schema
	Params  []paramSpec
}

// ParamLocation indicates where a parameter appears in the request.
type ParamLocation string

const (
	ParamPath  ParamLocation = "path"
	ParamQuery ParamLocation = "query"
)

// ParamType is the OpenAPI data type for a parameter.
type ParamType string

const (
	ParamString  ParamType = "string"
	ParamInteger ParamType = "integer"
)

// paramSpec describes a path or query parameter.
type paramSpec struct {
	Name        string
	In          ParamLocation
	Required    bool
	Type        ParamType
	Description string
}

func (p paramSpec) toOpenAPI() map[string]any {
	m := map[string]any{
		"name": p.Name, "in": string(p.In), "required": p.Required,
		"schema": map[string]any{"type": string(p.Type)},
	}
	if p.Description != "" {
		m["description"] = p.Description
	}
	return m
}

// schemaRegistry maps Go types to OpenAPI component schema names.
type schemaRegistry map[reflect.Type]string

func newSchemaRegistry() schemaRegistry {
	return schemaRegistry{
		reflect.TypeOf(model.Category{}):      "Category",
		reflect.TypeOf(model.Bookmark{}):      "Bookmark",
		reflect.TypeOf(CategoryCreateInput{}): "CategoryCreate",
		reflect.TypeOf(CategoryUpdateInput{}): "CategoryUpdate",
		reflect.TypeOf(BookmarkCreateInput{}): "BookmarkCreate",
		reflect.TypeOf(BookmarkUpdateInput{}): "BookmarkUpdate",
		reflect.TypeOf(BookmarkMoveInput{}):   "BookmarkMove",
	}
}

// schema generates an inline OpenAPI schema for the given Go type.
func (reg schemaRegistry) schema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		s := reg.schema(t.Elem())
		s["nullable"] = true
		return s
	}
	if t.Kind() == reflect.Slice {
		return map[string]any{"type": "array", "items": reg.schemaOrRef(t.Elem())}
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	}
	if t.Kind() != reflect.Struct {
		return map[string]any{"type": "string"}
	}

	props := map[string]any{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			continue
		}
		prop := reg.schemaOrRef(f.Type)
		if apiTag := f.Tag.Get("api"); apiTag != "" {
			for _, part := range strings.Split(apiTag, ",") {
				part = strings.TrimSpace(part)
				if part == "required" {
					required = append(required, jsonName)
				} else if strings.HasPrefix(part, "description=") {
					prop["description"] = strings.TrimPrefix(part, "description=")
				}
			}
		}
		props[jsonName] = prop
	}

	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

// schemaOrRef returns a $ref for registered types, otherwise inlines.
func (reg schemaRegistry) schemaOrRef(t reflect.Type) map[string]any {
	actual := t
	if t.Kind() == reflect.Ptr {
		actual = t.Elem()
	}
	if name, ok := reg[actual]; ok {
		s := map[string]any{"$ref": "#/components/schemas/" + name}
		if t.Kind() == reflect.Ptr {
			s["nullable"] = true
		}
		return s
	}
	return reg.schema(t)
}

// schemas generates all component schemas from registered types.
func (reg schemaRegistry) schemas() map[string]any {
	result := map[string]any{}
	for t, name := range reg {
		result[name] = reg.schema(t)
	}
	return result
}

func buildSpecFromRoutes(routes []apiRoute) map[string]any {
	reg := newSchemaRegistry()
	paths := map[string]any{}

	for _, r := range routes {
		oaPath := "/api" + strings.ReplaceAll(r.Path, ":id", "{id}")

		op := map[string]any{
			"summary": r.Summary,
			"tags":    []string{r.Tag},
		}

		if len(r.Params) > 0 {
			params := make([]map[string]any, len(r.Params))
			for i, p := range r.Params {
				params[i] = p.toOpenAPI()
			}
			op["parameters"] = params
		}

		if r.Input != nil {
			inputType := reflect.TypeOf(r.Input)
			var schema map[string]any
			if name, ok := reg[inputType]; ok {
				schema = map[string]any{"$ref": "#/components/schemas/" + name}
			} else {
				schema = reg.schema(inputType)
			}
			op["requestBody"] = map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{"schema": schema},
				},
			}
		}

		statusStr := fmt.Sprintf("%d", r.Status)
		if r.Output != nil {
			schema := reg.schemaOrRef(reflect.TypeOf(r.Output))
			op["responses"] = map[string]any{
				statusStr: map[string]any{
					"description": r.Summary,
					"content": map[string]any{
						"application/json": map[string]any{"schema": schema},
					},
				},
			}
		} else {
			op["responses"] = map[string]any{
				statusStr: map[string]any{"description": "Deleted"},
			}
		}

		method := strings.ToLower(r.Method)
		if existing, ok := paths[oaPath]; ok {
			existing.(map[string]any)[method] = op
		} else {
			paths[oaPath] = map[string]any{method: op}
		}
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Jumpgate API",
			"version": buildinfo.ServiceVersion(),
		},
		"paths": paths,
		"components": map[string]any{
			"schemas": reg.schemas(),
			"securitySchemes": map[string]any{
				"BearerAuth": map[string]any{
					"type":   "http",
					"scheme": "bearer",
				},
			},
		},
		"security": []map[string]any{
			{"BearerAuth": []string{}},
		},
	}
}
