package schema

import (
	"encoding/json"
	"reflect"
	"time"
)

type WithType uint

const (
	One WithType = iota
	Many
)

type IndexType string

const (
	PrimaryKey  IndexType = "PRIMARY KEY"
	UNIQUEKEY   IndexType = "UNIQUE KEY"
	INDEXKEY    IndexType = "KEY"
	FULLTEXTKEY IndexType = "FULLTEXT KEY"
)

type TimeType int64

var TimeReflectType = reflect.TypeOf(time.Time{})

var schemas = make(map[string]*Schema)

type Index struct {
	Priority int
	Field    *Field
}

type IndexList map[string][]Index

// Schema represents a table of database
type Schema struct {
	TablePrefix string
	Model       interface{}
	Value       reflect.Value
	Type        reflect.Type
	Name        string
	TableName   string
	FieldNames  []interface{}
	Fields      []*Field
	fieldMap    map[string]*Field
	Withs       map[string]*With
	PrimaryKey  *Field
	IndexKeys   IndexList
	UniqueKeys  IndexList
	FullKeys    IndexList
}

// GetField returns field by name
func (schema *Schema) GetField(fieldName string) *Field {
	return schema.fieldMap[fieldName]
}

// RecordValues Values return the values of dest's member variables
func (schema *Schema) RecordValues(omitEmpty, isUpdate bool) map[string]interface{} {
	fieldValues := make(map[string]interface{})

	for _, field := range schema.Fields {
		if schema.RecordValue(field, omitEmpty, isUpdate) {
			fieldValues[field.FieldName] = field.Value
		}
	}

	return fieldValues
}

func (schema *Schema) RecordValue(field *Field, omitEmpty, isUpdate bool) bool {
	if field.Raw {
		return false
	}

	if field.AutoIncrement {
		return false
	}

	if field.Name == "CreatedAt" ||
		field.Name == "UpdatedAt" {

		if isUpdate && field.Name == "CreatedAt" {
			return false
		}

		if field.DataType == Time {
			field.Value = time.Now().Format("2006-01-02 15:04:05.000")
			return true
		} else if field.Size == 32 && (field.DataType == Int || field.DataType == Uint) {
			field.Value = time.Now().Unix()
			return true
		} else if field.Size == 64 && field.DataType == Int {
			field.Value = time.Now().UnixMilli()
			return true
		} else if field.Size == 64 && field.DataType == Uint {
			field.Value = time.Now().UnixMicro()
			return true
		}
	}

	destVal := schema.Value.FieldByName(field.Name)
	value := destVal.Interface()
	if destVal.IsZero() {
		if omitEmpty {
			return false
		}

		if field.AutoIncrement ||
			field.DefaultValue == DefaultNull {
			return false
		}

		value = field.DefaultValue
	}

	if field.IsJson && field.DefaultValue == "" {
		value, _ = json.Marshal(value)
		value = string(value.([]byte))
	}

	field.Value = value
	return true
}

// Parse a struct to a Schema instance
func Parse(dest interface{}, dialect IDialect, tablePrefix string) *Schema {
	modelValue := reflect.Indirect(reflect.ValueOf(dest))
	modelType := modelValue.Type()

	if modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	model := reflect.New(modelType).Interface()
	var tableName string
	t, ok := model.(ITableName)
	if !ok {
		tableName = SnakeString(tablePrefix + modelType.Name())
	} else {
		tableName = t.TableName()
	}

	cacheKey := dialect.Name() + dialect.GetDSN() + tableName
	if schema, ok := schemas[cacheKey]; ok {
		schema.Value = modelValue
		return schema
	}

	schema := &Schema{
		TablePrefix: tablePrefix,
		Value:       modelValue,
		Type:        modelType,
		Model:       model,
		Name:        modelType.Name(),
		TableName:   tableName,
		fieldMap:    make(map[string]*Field),
		Withs:       make(map[string]*With),
		IndexKeys:   make(IndexList),
		UniqueKeys:  make(IndexList),
		FullKeys:    make(IndexList),
	}

	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)

		if p.Anonymous {
			defaultModelType := reflect.New(p.Type).Type().Elem()
			for j := 0; j < defaultModelType.NumField(); j++ {
				p1 := defaultModelType.Field(j)
				parseField(p1, dialect, schema, true)
			}
		} else {
			parseField(p, dialect, schema, false)
		}

	}

	schemas[cacheKey] = schema

	return schema
}

func MakeSlice(ModelType reflect.Type) reflect.Value {
	slice := reflect.MakeSlice(reflect.SliceOf(ModelType), 0, 20)
	results := reflect.New(slice.Type())
	results.Elem().Set(slice)
	return results
}
