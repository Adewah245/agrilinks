package render

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed shell.html
var shellSource string

//go:embed pages.json
var pageManifest []byte

type pageDefinition struct {
	Title       string
	Description string
	Styles      []string
	BodyClass   string
	Fields      []string
}

type projection map[string]projection

var pageDefinitions map[string]pageDefinition
var projections = map[string]projection{}
var shell = template.Must(template.New("react").Parse(shellSource))
var titles = map[string]*texttemplate.Template{}

func init() {
	if err := json.Unmarshal(pageManifest, &pageDefinitions); err != nil {
		panic(err)
	}
	for name, page := range pageDefinitions {
		titles[name] = texttemplate.Must(texttemplate.New(name).Parse(page.Title))
		root := projection{}
		for _, path := range page.Fields {
			node := root
			for _, key := range strings.Split(path, ".") {
				if key == "" {
					continue
				}
				if node[key] == nil {
					node[key] = projection{}
				}
				node = node[key]
			}
		}
		projections[name] = root
	}
}

// RenderPage serves the React shell with an explicitly limited page snapshot.
// Serializing handler models directly would expose fields such as PasswordHash.
func RenderPage(w http.ResponseWriter, name string, data any) error {
	page, ok := pageDefinitions[name]
	if !ok {
		return renderError(w, fmt.Errorf("unknown React page %q", name))
	}
	props, err := project(reflect.ValueOf(data), projections[name])
	if err != nil {
		return renderError(w, fmt.Errorf("React page %s: %w", name, err))
	}
	var title bytes.Buffer
	if err := titles[name].Execute(&title, props); err != nil {
		return renderError(w, err)
	}
	page.Title = strings.Join(strings.Fields(title.String()), " ")
	bootstrap := struct {
		Page string `json:"page"`
		Data any    `json:"data"`
	}{name, props}
	var output bytes.Buffer
	// The template's JavaScript context encodes and HTML-escapes JSON, including
	// </script>, so user values cannot terminate the bootstrap data element.
	err = shell.Execute(&output, struct {
		pageDefinition
		Bootstrap any
	}{page, bootstrap})
	if err != nil {
		return renderError(w, err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, err = w.Write(output.Bytes())
	return err
}

func renderError(w http.ResponseWriter, err error) error {
	http.Error(w, "This page could not be loaded. Please try again.", http.StatusInternalServerError)
	return err
}

func indirect(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func project(value reflect.Value, fields projection) (any, error) {
	value = indirect(value)
	if !value.IsValid() {
		return nil, nil
	}
	if value.Type() == reflect.TypeOf(time.Time{}) {
		return value.Interface(), nil
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		result := make([]any, value.Len())
		for i := range result {
			var err error
			result[i], err = project(value.Index(i), fields["*"])
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	if child, ok := fields["*"]; ok && value.Kind() == reflect.Map {
		result := map[string]any{}
		for _, key := range value.MapKeys() {
			item, err := project(value.MapIndex(key), child)
			if err != nil {
				return nil, err
			}
			result[fmt.Sprint(key.Interface())] = item
		}
		return result, nil
	}
	if value.Kind() != reflect.Struct && value.Kind() != reflect.Map {
		return value.Interface(), nil
	}
	if len(fields) == 0 {
		// Testing an object's presence does not authorize serializing its members.
		return true, nil
	}
	result := map[string]any{}
	for key, child := range fields {
		field, err := pageField(value, key)
		if err != nil {
			return nil, err
		}
		result[key], err = project(field, child)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
	}
	return result, nil
}

func pageField(value reflect.Value, key string) (reflect.Value, error) {
	if value.Kind() == reflect.Map {
		return value.MapIndex(reflect.ValueOf(key)), nil
	}
	if field := value.FieldByName(key); field.IsValid() {
		return field, nil
	}
	// Presentation methods and permissions are evaluated on the server.
	method := value.MethodByName(key)
	if !method.IsValid() {
		pointer := reflect.New(value.Type())
		pointer.Elem().Set(value)
		method = pointer.MethodByName(key)
	}
	if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 {
		return method.Call(nil)[0], nil
	}
	if method.IsValid() && key == "CanAdmin" {
		return reflect.ValueOf(map[string]bool{"revenue": method.Call([]reflect.Value{reflect.ValueOf("revenue")})[0].Bool()}), nil
	}
	return reflect.Value{}, fmt.Errorf("unknown page field %s.%s", value.Type(), key)
}
