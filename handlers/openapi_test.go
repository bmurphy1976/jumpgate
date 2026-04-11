package handlers

import (
	"dashboard/internal/buildinfo"
	"dashboard/model"
	"reflect"
	"strings"
	"testing"
)

func TestSchemaFieldCoverage(t *testing.T) {
	reg := newSchemaRegistry()
	for typ, name := range reg {
		schema := reg.schema(typ)
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Errorf("schema %s: expected properties map", name)
			continue
		}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			jsonTag := f.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName == "" {
				continue
			}
			if _, ok := props[jsonName]; !ok {
				t.Errorf("schema %s: field %q present in struct but missing from schema", name, jsonName)
			}
		}
	}
}

func TestSchemaRequiredFields(t *testing.T) {
	reg := newSchemaRegistry()

	tests := []struct {
		typ      any
		name     string
		required []string
	}{
		{CategoryCreateInput{}, "CategoryCreate", []string{"name"}},
		{BookmarkCreateInput{}, "BookmarkCreate", []string{"category_id"}},
		{BookmarkMoveInput{}, "BookmarkMove", []string{"category_id"}},
	}

	for _, tt := range tests {
		schema := reg.schema(reflect.TypeOf(tt.typ))
		required, _ := schema["required"].([]string)
		for _, field := range tt.required {
			found := false
			for _, r := range required {
				if r == field {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("schema %s: expected %q in required fields, got %v", tt.name, field, required)
			}
		}
	}
}

func TestSchemaTypeMapping(t *testing.T) {
	reg := newSchemaRegistry()

	tests := []struct {
		goType   reflect.Type
		oaType   string
		nullable bool
	}{
		{reflect.TypeOf(""), "string", false},
		{reflect.TypeOf(0), "integer", false},
		{reflect.TypeOf(true), "boolean", false},
		{reflect.TypeOf((*bool)(nil)), "boolean", true},
		{reflect.TypeOf((*string)(nil)), "string", true},
	}

	for _, tt := range tests {
		schema := reg.schema(tt.goType)
		if schema["type"] != tt.oaType {
			t.Errorf("Go type %v: expected OpenAPI type %q, got %v", tt.goType, tt.oaType, schema["type"])
		}
		if tt.nullable {
			if schema["nullable"] != true {
				t.Errorf("Go type %v: expected nullable=true", tt.goType)
			}
		}
	}
}

func TestSchemaRegistryRef(t *testing.T) {
	reg := newSchemaRegistry()

	// Registered type should produce $ref
	schema := reg.schemaOrRef(reflect.TypeOf(model.Category{}))
	if _, ok := schema["$ref"]; !ok {
		t.Error("expected $ref for registered type model.Category")
	}

	// Unregistered type should inline
	schema = reg.schemaOrRef(reflect.TypeOf(IconListOutput{}))
	if _, ok := schema["$ref"]; ok {
		t.Error("expected inline schema for unregistered type IconListOutput")
	}
	if schema["type"] != "object" {
		t.Errorf("expected type 'object' for inlined IconListOutput, got %v", schema["type"])
	}
}

func TestSchemaSliceOfRegisteredType(t *testing.T) {
	reg := newSchemaRegistry()

	schema := reg.schema(reflect.TypeOf([]model.Bookmark{}))
	if schema["type"] != "array" {
		t.Errorf("expected array type, got %v", schema["type"])
	}
	items, _ := schema["items"].(map[string]any)
	if items["$ref"] != "#/components/schemas/Bookmark" {
		t.Errorf("expected $ref to Bookmark, got %v", items)
	}
}

func TestBuildSpecStructure(t *testing.T) {
	oldRelease := buildinfo.ReleaseVersion
	oldCommit := buildinfo.Commit
	buildinfo.ReleaseVersion = "2026.04.0"
	buildinfo.Commit = "04fc78e"
	t.Cleanup(func() {
		buildinfo.ReleaseVersion = oldRelease
		buildinfo.Commit = oldCommit
	})

	h := &APIHandler{}
	routes := dataRoutes(h)
	spec := buildSpecFromRoutes(routes)

	if spec["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %v", spec["openapi"])
	}
	info := spec["info"].(map[string]any)
	if info["version"] != "2026.04.0+04fc78e" {
		t.Errorf("expected stamped version, got %v", info["version"])
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths map")
	}

	expectedPaths := []string{
		"/api/categories",
		"/api/categories/{id}",
		"/api/bookmarks/{id}",
		"/api/bookmarks/search",
		"/api/bookmarks/{id}/move",
		"/api/icons",
	}
	for _, p := range expectedPaths {
		if _, ok := paths[p]; !ok {
			t.Errorf("missing path %s", p)
		}
	}

	components, _ := spec["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	expectedSchemas := []string{
		"Category", "Bookmark",
		"CategoryCreate", "CategoryUpdate",
		"BookmarkCreate", "BookmarkUpdate", "BookmarkMove",
	}
	for _, s := range expectedSchemas {
		if _, ok := schemas[s]; !ok {
			t.Errorf("missing schema %s", s)
		}
	}
}

func TestBuildSpecRouteHasInputSchema(t *testing.T) {
	h := &APIHandler{}
	routes := dataRoutes(h)
	spec := buildSpecFromRoutes(routes)
	paths := spec["paths"].(map[string]any)

	// POST /api/categories should have a requestBody referencing CategoryCreate
	catPath := paths["/api/categories"].(map[string]any)
	postOp := catPath["post"].(map[string]any)
	reqBody, ok := postOp["requestBody"].(map[string]any)
	if !ok {
		t.Fatal("POST /api/categories missing requestBody")
	}
	content := reqBody["content"].(map[string]any)
	jsonContent := content["application/json"].(map[string]any)
	schema := jsonContent["schema"].(map[string]any)
	if schema["$ref"] != "#/components/schemas/CategoryCreate" {
		t.Errorf("expected $ref to CategoryCreate, got %v", schema)
	}
}

func TestBuildSpecRouteHasOutputSchema(t *testing.T) {
	h := &APIHandler{}
	routes := dataRoutes(h)
	spec := buildSpecFromRoutes(routes)
	paths := spec["paths"].(map[string]any)

	// GET /api/categories should return array of Category
	catPath := paths["/api/categories"].(map[string]any)
	getOp := catPath["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	jsonContent := content["application/json"].(map[string]any)
	schema := jsonContent["schema"].(map[string]any)
	if schema["type"] != "array" {
		t.Errorf("expected array type for GET /api/categories response, got %v", schema["type"])
	}
	items := schema["items"].(map[string]any)
	if items["$ref"] != "#/components/schemas/Category" {
		t.Errorf("expected $ref to Category, got %v", items)
	}
}

func TestBuildSpecDeleteHasNoOutput(t *testing.T) {
	h := &APIHandler{}
	routes := dataRoutes(h)
	spec := buildSpecFromRoutes(routes)
	paths := spec["paths"].(map[string]any)

	catIDPath := paths["/api/categories/{id}"].(map[string]any)
	deleteOp := catIDPath["delete"].(map[string]any)
	responses := deleteOp["responses"].(map[string]any)
	resp204 := responses["204"].(map[string]any)
	if _, hasContent := resp204["content"]; hasContent {
		t.Error("DELETE response should not have content")
	}
	if resp204["description"] != "Deleted" {
		t.Errorf("expected description 'Deleted', got %v", resp204["description"])
	}
}
