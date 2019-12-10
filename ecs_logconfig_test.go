package awsecs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"testing"
)

func TestAlterLogConfigurationLogDriverOptions(t *testing.T) {
	optionToDelete := "myDeletedOption"
	logConfig := ecs.LogConfiguration{
		LogDriver: aws.String("mydriver"),
		Options: map[string]*string{
			"myUntouchedOption": aws.String("myUntouchedOptionOriginalValue"),
			"myTouchedOption":   aws.String("myTouchedOptionOriginalValue"),
			optionToDelete:      aws.String("myDeletedOptionOriginalValue"),
		},
		SecretOptions: []*ecs.Secret{},
	}
	alterLogConfigurationLogDriverOptions(logConfig, map[string]map[string]string{
		"mydriver": {
			"myTouchedOption": "myTouchedOptionNewValue",
			optionToDelete:    EnvKnockOutValue,
		},
	})
	if *logConfig.LogDriver != "mydriver" {
		t.Fail()
	}
	if *logConfig.Options["myUntouchedOption"] != "myUntouchedOptionOriginalValue" {
		t.Fail()
	}
	if *logConfig.Options["myTouchedOption"] != "myTouchedOptionNewValue" {
		t.Fail()
	}
	if len(logConfig.Options) != 2 {
		t.Fail()
	}
}

func TestDeleteLogConfigurationLogDriverOptions(t *testing.T) {
	optionToDelete1 := "myDeletedOption1"
	optionToDelete2 := "myDeletedOption2"
	logConfig := ecs.LogConfiguration{
		LogDriver: aws.String("olddriver"),
		Options: map[string]*string{
			"myUntouchedOption": aws.String("myUntouchedOptionOriginalValue"),
			optionToDelete1:     aws.String("myDeletedOptionOriginalValue1"),
			optionToDelete2:     aws.String("myDeletedOptionOriginalValue2"),
		},
		SecretOptions: []*ecs.Secret{},
	}
	alterLogConfigurationLogDriverOptions(logConfig, map[string]map[string]string{
		"olddriver": {
			"":              "ignored",
			optionToDelete1: "",
		},
		"newdriver": {
			"myOptionToAdd": "myOptionToAddValue",
			optionToDelete2: "",
		},
	})
	if *logConfig.Options["myUntouchedOption"] != "myUntouchedOptionOriginalValue" {
		t.Fail()
	}
	if *logConfig.Options["myOptionToAdd"] != "myOptionToAddValue" {
		t.Fail()
	}
	if len(logConfig.Options) != 2 {
		t.Fail()
	}
}
