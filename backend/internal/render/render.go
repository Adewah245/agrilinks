package render

import (
	"bytes"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func commaNumber(value float64, decimals int) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	s := strconv.FormatFloat(value, 'f', decimals, 64)
	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	sign := ""
	if strings.HasPrefix(whole, "-") {
		sign, whole = "-", whole[1:]
	}
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if decimals == 0 {
		return sign + whole
	}
	return sign + whole + "." + parts[1]
}

func money(value float64) string  { return commaNumber(value, 2) }
func money0(value float64) string { return commaNumber(value, 0) }
func hasCoordinates(latitude, longitude float64) bool {
	return latitude != 0 && longitude != 0
}

func loadTemplates() *template.Template {
	funcs := template.FuncMap{
		"money":          money,
		"money0":         money0,
		"comma":          func(v int) string { return commaNumber(float64(v), 0) },
		"hasCoordinates": hasCoordinates,
	}
	for _, pattern := range []string{"../frontend/pages/*.html", "../../frontend/pages/*.html", "../../../frontend/pages/*.html", "frontend/pages/*.html"} {
		if matches, err := os.ReadDir("."); err == nil && matches != nil {
			if templates, parseErr := template.New("base").Funcs(funcs).ParseGlob(pattern); parseErr == nil {
				return templates
			}
		}
	}
	return template.Must(template.New("base").Funcs(funcs).ParseGlob("../../../frontend/pages/*.html"))
}

var TMPL = loadTemplates()

func RenderTemplates(w http.ResponseWriter, name string, data any) error {
	// Render before writing to the HTTP response. A template error after a
	// partial write prevents callers from returning a correct error response
	// and produces a "superfluous response.WriteHeader" server warning.
	var output bytes.Buffer
	if err := TMPL.ExecuteTemplate(&output, name, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write(output.Bytes())
	return err
}
