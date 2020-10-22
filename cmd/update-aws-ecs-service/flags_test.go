package main

import (
	"reflect"
	"testing"
)

func TestMapMapMapFlag_Set(t *testing.T) {
	actualStruct := mapMapMapFlag{}
	if err := actualStruct.Set("container1=awslogs=region=us-west-2"); err != nil {
		t.Fatal(err)
	}
	if err := actualStruct.Set("container1=awslogs=loggroup=group1"); err != nil {
		t.Fatal(err)
	}
	if err := actualStruct.Set("container2=awslogs=="); err != nil {
		t.Fatal(err)
	}
	if err := actualStruct.Set("container2=fluentd=option1=value1"); err != nil {
		t.Fatal(err)
	}
	var expectedStruct mapMapMapFlag = map[string]map[string]map[string]string{
		"container1": {
			"awslogs": {
				"region":   "us-west-2",
				"loggroup": "group1",
			},
		},
		"container2": {
			"awslogs": {
				"": "",
			},
			"fluentd": {
				"option1": "value1",
			},
		},
	}
	if !reflect.DeepEqual(expectedStruct, actualStruct) {
		t.Fatal()
	}
}
