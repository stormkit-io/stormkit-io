package utils

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

const NoFilter = ""

func getFields(s interface{}, prefix string, excludedFields ...string) []string {
	var names []string
	t := reflect.TypeOf(s)

	isExcluded := func(item string) bool {
		for _, excluded := range excludedFields {
			if item == excluded {
				return true
			}
		}
		return false
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		colName := field.Tag.Get("db")
		if isExcluded(colName) {
			continue
		}
		if colName != "" && !isExcluded(colName) {
			if prefix != "" {
				colName = fmt.Sprintf("%s.%s", prefix, colName)
			}

			names = append(names, colName)
		}
	}
	return names
}

func buildQuery(text string, data map[string]interface{}) string {
	var t *template.Template
	incFunc := func(num int) int {
		return num + 1
	}

	t = template.Must(template.New("query").Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
		"inc":         incFunc,
	}).Parse(text))

	var wr bytes.Buffer
	if err := t.Execute(&wr, data); err != nil {
		slog.Error("could not apply sql query data: %w", err)
		return ""
	}

	query := wr.String()
	return strings.TrimSpace(query)
}

// GenerateArray generates numbered parameters for an array in the
// following format:
//
// $1, $2, $3 ...
//
// The first argument will tell this function from which number to start.
// The second argument is the number of parameters this function should print.
func GenerateArray(initialCounter, values int) string {
	retval := []string{}

	for value := initialCounter + 1; value < initialCounter+values+1; value++ {
		retval = append(retval, fmt.Sprintf("$%d", value))
	}

	return strings.Join(retval, ", ")
}

// Generates place holder string for insert statement in posgresql
func GenerateValues(numberOfFields, rows int) string {
	var builder strings.Builder

	for row := 0; row < rows; row++ {
		builder.WriteString("(")
		for field := 0; field < numberOfFields; field++ {
			if field > 0 {
				builder.WriteString(", ")
			}
			index := row*numberOfFields + field + 1
			builder.WriteString(fmt.Sprintf("$%d", index))
		}
		if row < rows-1 {
			builder.WriteString("),")
		} else {
			builder.WriteString(")")
		}
	}

	return builder.String()
}

// GenerateRecordRow iterates for a given numberOfParameters time and creates
// a map in the following format:
//
//	{
//	  "p1": x,
//	  "p2": x + 1,
//	}
//
// This map can be used in query templates.
func GenerateRecordRow(numberOfParameters int, counter *int) map[string]int {
	row := map[string]int{}

	for i := 0; i < numberOfParameters; i++ {
		row["p"+strconv.Itoa(i+1)] = *counter + i + 1
	}

	*counter = *counter + numberOfParameters

	return row
}

func QInsert(tableName string, fields ...string) string {
	args := make(map[string]interface{})
	args["fields"] = fields
	args["fieldsSize"] = len(fields) - 1
	args["tableName"] = tableName
	query :=
		`INSERT INTO {{.tableName}} ({{ StringsJoin .fields ", " }}) ` +
			"VALUES (" +
			/**/ "{{range $i, $a := .fields}}" +
			/**/ "${{inc $i}}" +
			/**/ "{{ if ne $i $.fieldsSize }}, {{end}}" +
			/**/ "{{end}}" +
			");"

	return buildQuery(query, args)
}

func QUpdate(tableName string, condition string, fields ...string) string {
	args := make(map[string]interface{})
	args["fields"] = fields
	args["fieldsSize"] = len(fields) - 1
	args["tableName"] = tableName

	if condition != NoFilter {
		args["condition"] = condition
	}

	query :=
		"UPDATE {{.tableName}} SET " +
			/**/ "{{range $i, $a := .fields}}" +
			/**/ "{{$a}} = ${{inc $i}}" +
			/**/ "{{ if ne $i $.fieldsSize}}, {{end}}" +
			/**/ "{{end}}" +
			"{{ if .condition }}" +
			/**/ " WHERE {{.condition}}" +
			"{{end}};"

	return buildQuery(query, args)
}

// all will print all
func QSelect(tableName string, condition string, fields ...string) string {
	args := make(map[string]interface{})
	args["fields"] = strings.Join(fields, ",")
	args["tableName"] = tableName

	if condition != NoFilter {
		args["condition"] = condition
	}

	query :=
		"SELECT {{.fields}} FROM {{.tableName}}" +
			"{{ if .condition }}" +
			" WHERE {{.condition}}" +
			"{{ end }};"

	return buildQuery(query, args)
}

func QSelectAllExcept(givenType interface{}, condition string, excludedFields ...string) string {
	args := make(map[string]interface{})
	fields := getFields(givenType, "", excludedFields...)
	// sort.Strings(fields)
	args["fields"] = strings.Join(fields, ",")

	args["tableName"] = reflect.ValueOf(givenType).MethodByName("TableName").Call([]reflect.Value{})[0]

	if condition != NoFilter {
		args["condition"] = condition
	}

	query :=
		"SELECT {{.fields}} FROM {{.tableName}}" +
			"{{ if .condition }}" +
			" WHERE {{.condition}}" +
			"{{ end }};"

	return buildQuery(query, args)
}
