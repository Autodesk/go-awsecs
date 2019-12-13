package main

import "fmt"

type mapMapMapFlag map[string]map[string]map[string]string

func (kvs *mapMapMapFlag) String() string {
	return fmt.Sprintf("%v", *kvs)
}

func (kvs mapMapMapFlag) Set(value string) error {
	key, value := keyEqValue(value)
	valueKey, value := keyEqValue(value)
	valueValueKey, value := keyEqValue(value)
	if kvs[key] == nil {
		kvs[key] = map[string]map[string]string{}
	}
	if kvs[key][valueKey] == nil {
		kvs[key][valueKey] = map[string]string{}
	}
	kvs[key][valueKey][valueValueKey] = value
	return nil
}
