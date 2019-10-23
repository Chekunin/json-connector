package json_connector

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"reflect"
	"strings"
)

type JsonConnector struct {
	path         string
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
	pathToFile        string
}

func NewJsonConnector(model interface{}, pathToFile string) *JsonConnector {
	return &JsonConnector{
		path:         pathToFile,
		model:        model,
		dependencies: make(map[string]dependency),
	}
}

func (jc *JsonConnector) AddDependency(fieldName string, pathToFile string) *JsonConnector {
	dep := dependency{
		fieldName:  fieldName,
		pathToFile: pathToFile,
	}
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
	data, err := ioutil.ReadFile(jc.path)
	if err != nil {
		return err
	}

	if len(jc.filters) > 1 {
		return errors.New("max number of where-conditions is one")
	}

	filterStr := jc.getFilterStr()
	if filterStr != "" {
		resultData := gjson.GetBytes(data, filterStr)
		err := json.Unmarshal([]byte(resultData.String()), &jc.model)
		if err != nil {
			return err
		}
	} else {
		err := json.Unmarshal(data, &jc.model)
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
			for _, v := range jc.dependencies {
				err := jc.fillDependencyField(elemValue, v)
				if err != nil {
					return err
				}
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
	switch elemValue.FieldByName(dep.localFKFieldName).Kind() {
	case reflect.Int:
		fkValInt := elemValue.FieldByName(dep.localFKFieldName).Interface().(int)
		data2, err := ioutil.ReadFile(dep.pathToFile)
		if err != nil {
			return err
		}
		jsonTagVal, ok := getTagValueInFieldWithName(
			elemValue.FieldByName(dep.fieldName).Interface(),
			dep.remotePKFieldName,
			"json")
		if !ok {
			jsonTagVal = dep.remotePKFieldName
		}
		filterStr2 := fmt.Sprintf("#(%s=%d)", jsonTagVal, fkValInt)
		res2 := gjson.GetBytes(data2, filterStr2)
		err = json.Unmarshal(data2[res2.Index:res2.Index+len(res2.Raw)], newFieldObjPtr.Interface())
		if err != nil {
			return err
		}

		elemValue.FieldByName(dep.fieldName).Set(reflect.Indirect(newFieldObjPtr))
	}
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
			filterStr += fmt.Sprintf("%s%s", v.fieldName, v.operation)
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

func getFieldValueWithName(model interface{}, fieldName string) interface{} {
	r := reflect.ValueOf(model)
	f := reflect.Indirect(r).FieldByName(fieldName)
	return f.Interface()
}
