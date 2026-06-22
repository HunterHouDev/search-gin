package utils

import (
	"encoding/json"
	"reflect"
)

func InterfaceToMap(setting interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	refType := reflect.TypeOf(setting)
	refValue := reflect.ValueOf(setting)
	for i := 0; i < refValue.NumField(); i++ {
		data[refType.Field(i).Name] = refValue.Field(i)
	}
	return data
}

// InterfaceFields 获取结构体字段名列表
func InterfaceFields(setting interface{}) []string {

	refType := reflect.TypeOf(setting)
	data := make([]string, refType.NumField())
	for i := 0; i < refType.NumField(); i++ {
		data[i] = refType.Field(i).Name
	}
	return data
}

func FieldsMapToStruct(setting interface{}, valueMap map[string]interface{}) error {
	jsn, _ := json.Marshal(valueMap)
	err := json.Unmarshal(jsn, setting)
	if err != nil {
		return err
	}
	return nil
}
