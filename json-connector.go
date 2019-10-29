package json_connector

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"reflect"
	"strings"
)

type JsonConnector struct {
	data         []byte
	model        interface{}
	dependencies map[string]dependency
	filters      []filter
}

type filter struct {
	fieldName string
	operation string
	value     interface{}
}

type dependency struct {
	fieldName         string
	localFKFieldName  string
	remotePKFieldName string
	data              []byte
}

func NewJsonConnector(model interface{}, data []byte) *JsonConnector {
	return &JsonConnector{
		data:         data,
		model:        model,
		dependencies: make(map[string]dependency),
	}
}

func (jc *JsonConnector) AddDependency(fieldName string, data []byte) *JsonConnector {
	dep := dependency{
		fieldName: fieldName,
		data:      data,
	}
	if !strings.Contains(fieldName, ".") {
		if tag, ok := getTagValueInFieldWithName(jc.model, fieldName, "jc"); ok {
			if len(strings.Split(tag, ",")) < 2 {
				panic(errors.New("need two fieldnames in tag jc of field " + fieldName))
			}
			val1 := strings.Split(tag, ",")[0]
			val2 := strings.Split(tag, ",")[1]
			dep.localFKFieldName = val1
			dep.remotePKFieldName = val2
		} else {
			panic(fmt.Sprintf("tag jc in field %s doesn't filled", fieldName))
		}
	}
	jc.dependencies[fieldName] = dep
	return jc
}

func (jc *JsonConnector) Where(fieldName string, operation string, value interface{}) *JsonConnector {
	jc.filters = append(jc.filters, filter{
		fieldName: fieldName,
		operation: operation,
		value:     value,
	})
	return jc
}

func (jc *JsonConnector) Unmarshal() error {
	if len(jc.filters) > 1 {
		return errors.New("max number of where-conditions is one")
	}

	filterStr := jc.getFilterStr()
	if filterStr != "" {
		resultData := gjson.GetBytes(jc.data, filterStr)
		if len(resultData.Raw) == 0 {
			return nil
		}
		err := json.Unmarshal([]byte(resultData.String()), &jc.model)
		if err != nil {
			return err
		}
	} else {
		err := json.Unmarshal(jc.data, &jc.model)
		if err != nil {
			return err
		}
	}

	modelType := reflect.TypeOf(jc.model)
	if reflect.TypeOf(jc.model).Kind() != reflect.Ptr {
		return errors.New("model must be a pointer")
	}
	modelType = modelType.Elem()
	switch modelType.Kind() {
	case reflect.Slice:
		modelValue := reflect.Indirect(reflect.ValueOf(jc.model))
		for i := 0; i < modelValue.Len(); i++ {
			elemValue := reflect.Indirect(modelValue.Index(i))
			err := jc.fillDependencyFields(elemValue, jc.dependencies)
			if err != nil {
				return err
			}
		}
	case reflect.Struct:
		elemValue := reflect.ValueOf(jc.model)
		err := jc.fillDependencyFields(elemValue, jc.dependencies)
		if err != nil {
			return err
		}
	case reflect.Ptr:
		elemValue := reflect.ValueOf(jc.model).Elem().Elem()
		err := jc.fillDependencyFields(elemValue, jc.dependencies)
		if err != nil {
			return err
		}
	}

	return nil
}

func (jc *JsonConnector) fillDependencyFields(elemValue reflect.Value, deps map[string]dependency) error {
	for _, v := range deps {
		if strings.Contains(v.fieldName, ".") {
			continue
		}
		if err := jc.fillDependencyField(elemValue, v); err != nil {
			return err
		}
	}
	return nil
}
func (jc *JsonConnector) fillDependencyField(elemValue reflect.Value, dep dependency) error {
	fieldValue := elemValue.FieldByName(dep.fieldName)
	if !fieldValue.IsValid() {
		return errors.New(fmt.Sprintf("no field %s in struct %s",
			dep.fieldName,
			elemValue.Type().String()))
	}

	fieldType := fieldValue.Type()
	newFieldObjPtr := reflect.New(fieldType)
	tempJc := NewJsonConnector(newFieldObjPtr.Interface(), dep.data)
	switch elemValue.FieldByName(dep.localFKFieldName).Kind() {
	case reflect.Int:
		fkValInt := elemValue.FieldByName(dep.localFKFieldName).Interface().(int)
		tempJc = tempJc.Where(dep.remotePKFieldName, "=", fkValInt)
	case reflect.String:
		skValStr := elemValue.FieldByName(dep.localFKFieldName).Interface().(string)
		tempJc = tempJc.Where(dep.remotePKFieldName, "=", fmt.Sprintf("\"%s\"", skValStr))
	case reflect.Float64:
		skValFlt64 := elemValue.FieldByName(dep.localFKFieldName).Interface().(float64)
		tempJc = tempJc.Where(dep.remotePKFieldName, "=", skValFlt64)
	case reflect.Float32:
		skValFlt32 := elemValue.FieldByName(dep.localFKFieldName).Interface().(float32)
		tempJc = tempJc.Where(dep.remotePKFieldName, "=", skValFlt32)
	case reflect.Uint:
		skValUint := elemValue.FieldByName(dep.localFKFieldName).Interface().(uint)
		tempJc = tempJc.Where(dep.remotePKFieldName, "=", skValUint)
	}
	for _, v1 := range jc.dependencies {
		v1Arr := strings.Split(v1.fieldName, ".")
		if len(v1Arr) > 1 && v1Arr[0] == dep.fieldName {
			tempJc = tempJc.AddDependency(strings.Join(v1Arr[1:], "."), v1.data)
		}
	}
	if err := tempJc.Unmarshal(); err != nil {
		return err
	}
	elemValue.FieldByName(dep.fieldName).Set(reflect.Indirect(newFieldObjPtr))

	return nil
}

func (jc *JsonConnector) getFilterStr() string {
	var filterStr string
	if len(jc.filters) != 0 {
		for _, v := range jc.filters {
			if filterStr != "" {
				filterStr += "#."
			}
			filterStr += "#("
			fieldNameInJson, ok := getTagValueInFieldWithName(jc.model, v.fieldName, "json")
			if !ok {
				fieldNameInJson = v.fieldName
			}
			fieldNameInJson = strings.Split(fieldNameInJson, ",")[0]
			filterStr += fmt.Sprintf("%s%s", fieldNameInJson, v.operation)
			switch v.value.(type) {
			case string:
				filterStr += fmt.Sprintf("\"%s\"", v.value)
			case int:
				filterStr += fmt.Sprintf("%d", v.value)
			case bool:
				filterStr += fmt.Sprintf("%t", v.value)
			default:
				panic(fmt.Sprintf("I don't know type %T", v.value))
			}
			filterStr += ")"
		}
		if reflect.TypeOf(jc.model).Elem().Kind() == reflect.Slice ||
			reflect.TypeOf(jc.model).Elem().Kind() == reflect.Array {
			filterStr += "#"
		}
	}
	return filterStr
}

func getTagValueInFieldWithName(model interface{}, fieldName string, tagName string) (string, bool) {
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Slice ||
		t.Kind() == reflect.Array ||
		t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if f, ok := t.FieldByName(fieldName); ok {
		tagValue := f.Tag.Get(tagName)
		return tagValue, true
	} else {
		return "", false
	}
}
