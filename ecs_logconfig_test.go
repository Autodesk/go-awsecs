package awsecs

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/sergi/go-diff/diffmatchpatch"
	"reflect"
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
	logConfig = alterLogConfigurationLogDriverOptions(logConfig, map[string]map[string]string{
		"mydriver": {
			"myTouchedOption": "myTouchedOptionNewValue",
			optionToDelete:    EnvKnockOutValue,
		},
	})
	if *logConfig.LogDriver != "mydriver" {
		t.Fatal()
	}
	if *logConfig.Options["myUntouchedOption"] != "myUntouchedOptionOriginalValue" {
		t.Fatal()
	}
	if *logConfig.Options["myTouchedOption"] != "myTouchedOptionNewValue" {
		t.Fatal()
	}
	if len(logConfig.Options) != 2 {
		t.Fatal()
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
	logConfig = alterLogConfigurationLogDriverOptions(logConfig, map[string]map[string]string{
		"olddriver": {
			"":              "ignored",
			optionToDelete1: "",
		},
		"newdriver": {
			"myOptionToAdd": "myOptionToAddValue",
			optionToDelete2: "",
		},
	})
	if *logConfig.LogDriver != "newdriver" {
		t.Fatal()
	}
	if *logConfig.Options["myUntouchedOption"] != "myUntouchedOptionOriginalValue" {
		t.Fatal()
	}
	if *logConfig.Options["myOptionToAdd"] != "myOptionToAddValue" {
		t.Fatal()
	}
	if len(logConfig.Options) != 2 {
		t.Fatal()
	}
}

func TestAlterLogConfigurationLogDriverSecrets(t *testing.T) {
	optionToDelete := "myDeletedOption"
	logConfig := ecs.LogConfiguration{
		LogDriver: aws.String("mydriver"),
		Options:   map[string]*string{},
		SecretOptions: []*ecs.Secret{
			{
				Name:      aws.String("myUntouchedSecretOption"),
				ValueFrom: aws.String("myUntouchedSecretOptionOriginalValue"),
			},
			{
				Name:      aws.String("myTouchedSecretOption"),
				ValueFrom: aws.String("myTouchedSecretOptionOriginalValue"),
			},
			{
				Name:      aws.String(optionToDelete),
				ValueFrom: aws.String("myDeletedSecretOption"),
			},
		},
	}
	logConfig = alterLogConfigurationLogDriverSecrets(logConfig, map[string]map[string]string{
		"mydriver": {
			"myTouchedSecretOption": "myTouchedSecretOptionNewValue",
			"myAddedSecretOption":   "myAddedSecretOptionNewValue",
			optionToDelete:          EnvKnockOutValue,
		},
	})
	if *logConfig.LogDriver != "mydriver" {
		t.Fatal()
	}
	for _, secretOption := range logConfig.SecretOptions {
		if *secretOption.Name == "myUntouchedSecretOption" {
			if *secretOption.ValueFrom != "myUntouchedSecretOptionOriginalValue" {
				t.Fatal()
			}
		}
		if *secretOption.Name == "myTouchedSecretOption" {
			if *secretOption.ValueFrom != "myTouchedSecretOptionNewValue" {
				t.Fatal()
			}
		}
		if *secretOption.Name == optionToDelete {
			t.Fatal()
		}
		if *secretOption.Name == "myAddedSecretOption" {
			if *secretOption.ValueFrom != "myAddedSecretOptionNewValue" {
				t.Fatal()
			}
		}
	}
	if len(logConfig.SecretOptions) != 3 {
		t.Fatal()
	}
}

func TestDeleteLogConfigurationLogDriverSecrets(t *testing.T) {
	optionToDelete1 := "myDeletedOption1"
	optionToDelete2 := "myDeletedOption2"
	logConfig := ecs.LogConfiguration{
		LogDriver: aws.String("myolddriver"),
		Options:   map[string]*string{},
		SecretOptions: []*ecs.Secret{
			{
				Name:      aws.String("myUntouchedSecretOption"),
				ValueFrom: aws.String("myUntouchedSecretOptionOriginalValue"),
			},
			{
				Name:      aws.String(optionToDelete1),
				ValueFrom: aws.String("myDeletedSecretOption1"),
			},
			{
				Name:      aws.String(optionToDelete2),
				ValueFrom: aws.String("myDeletedSecretOption2"),
			},
		},
	}
	logConfig = alterLogConfigurationLogDriverSecrets(logConfig, map[string]map[string]string{
		"myolddriver": {
			"":              "ignore",
			optionToDelete1: EnvKnockOutValue,
		},
		"mynewdriver": {
			"myAddedSecretOption": "myAddedSecretOptionNewValue",
			optionToDelete2:       EnvKnockOutValue,
		},
	})
	if *logConfig.LogDriver != "mynewdriver" {
		t.Fatal()
	}
	for _, secretOption := range logConfig.SecretOptions {
		if *secretOption.Name == "myUntouchedSecretOption" {
			if *secretOption.ValueFrom != "myUntouchedSecretOptionOriginalValue" {
				t.Fatal()
			}
		}
		if *secretOption.Name == optionToDelete1 {
			t.Fatal()
		}
		if *secretOption.Name == optionToDelete2 {
			t.Fatal()
		}
		if *secretOption.Name == "myAddedSecretOption" {
			if *secretOption.ValueFrom != "myAddedSecretOptionNewValue" {
				t.Fatal()
			}
		}
	}
	if len(logConfig.SecretOptions) != 2 {
		t.Fatal()
	}
}

func TestAlterLogConfigurationsExistingOption(t *testing.T) {
	testObject := ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Name: aws.String("container1"),
			},
			{
				Name: aws.String("container2"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver2"),
					Options: map[string]*string{
						"driver2Option1": aws.String("driver2Value1"),
						"driver2Option2": aws.String("driver2Value2"),
					},
				},
			},
			{
				Name: aws.String("container3"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver3"),
					Options: map[string]*string{
						"driver3KeepOption1":   aws.String("driver3Value1"),
						"driver3KeepOption2":   aws.String("driver3Value2"),
						"driver3RemoveOption1": aws.String("driver3Value3"),
					},
				},
			},
			{
				Name: aws.String("container4"),
			},
			{
				Name: aws.String("container5"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver5"),
					Options: map[string]*string{
						"driver5Option1": aws.String("driver5Value1"),
						"driver5Option2": aws.String("driver5Value2"),
					},
				},
			},
			{
				Name: aws.String("container6"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver6"),
					Options: map[string]*string{
						"driver6Option1":       aws.String("driver6Value1"),
						"driver6Option2":       aws.String("driver6Value2"),
						"driver6UpdateOption1": aws.String("driver6OldValue1"),
					},
				},
			},
			{
				Name: aws.String("container7"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("oldDriver7"),
					Options: map[string]*string{
						"driver7KeepOption1": aws.String("driver7Value1"),
						"driver7KeepOption2": aws.String("driver7Value2"),
					},
				},
			},
			{
				Name: aws.String("container8"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver8"),
					Options: map[string]*string{
						"driver8Option1": aws.String("driver8Value1"),
						"driver8Option2": aws.String("driver8Value2"),
					},
				},
			},
			{
				Name: aws.String("container9"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver9"),
					Options: map[string]*string{
						"driver9Option1": aws.String("driver9Value1"),
						"driver9Option2": aws.String("driver9Value2"),
					},
				},
			},
		},
	}
	actualResult := alterLogConfigurations(testObject, map[string]map[string]map[string]string{
		"container3": {
			"driver3": {
				"driver3RemoveOption1": "",
			},
		},
		"container4": {
			"driver4": {
				"driver4Option1": "driver4Value1",
				"driver4Option2": "driver4Value2",
			},
		},
		"container5": {
			"driver5": {
				"driver5Option3": "driver5Value3",
			},
		},
		"container6": {
			"driver6": {
				"driver6UpdateOption1": "driver6NewValue1",
			},
		},
		"container7": {
			"oldDriver7": {
				"": "",
			},
			"newDriver7": {
				"newDriver7NewOption1": "newDriver7Value1",
			},
		},
		"container8": {
			"newDriver8": {
				"newDriver8Option1": "newDriver8Value1",
			},
		},
		"container9": {
			"driver9": {
				"": "",
			},
		},
	}, nil)
	expectedResult := []*ecs.ContainerDefinition{
		{
			Name: aws.String("container1"),
		},
		{
			Name: aws.String("container2"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver2"),
				Options: map[string]*string{
					"driver2Option1": aws.String("driver2Value1"),
					"driver2Option2": aws.String("driver2Value2"),
				},
			},
		},
		{
			Name: aws.String("container3"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver3"),
				Options: map[string]*string{
					"driver3KeepOption1": aws.String("driver3Value1"),
					"driver3KeepOption2": aws.String("driver3Value2"),
				},
			},
		},
		{
			Name: aws.String("container4"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver4"),
				Options: map[string]*string{
					"driver4Option1": aws.String("driver4Value1"),
					"driver4Option2": aws.String("driver4Value2"),
				},
			},
		},
		{
			Name: aws.String("container5"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver5"),
				Options: map[string]*string{
					"driver5Option1": aws.String("driver5Value1"),
					"driver5Option2": aws.String("driver5Value2"),
					"driver5Option3": aws.String("driver5Value3"),
				},
			},
		},
		{
			Name: aws.String("container6"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver6"),
				Options: map[string]*string{
					"driver6Option1":       aws.String("driver6Value1"),
					"driver6Option2":       aws.String("driver6Value2"),
					"driver6UpdateOption1": aws.String("driver6NewValue1"),
				},
			},
		},
		{
			Name: aws.String("container7"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("newDriver7"),
				Options: map[string]*string{
					"driver7KeepOption1":   aws.String("driver7Value1"),
					"driver7KeepOption2":   aws.String("driver7Value2"),
					"newDriver7NewOption1": aws.String("newDriver7Value1"),
				},
			},
		},
		{
			Name: aws.String("container8"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver8"),
				Options: map[string]*string{
					"driver8Option1": aws.String("driver8Value1"),
					"driver8Option2": aws.String("driver8Value2"),
				},
			},
		},
		{
			Name: aws.String("container9"),
		},
	}
	if !reflect.DeepEqual(actualResult.ContainerDefinitions, expectedResult) {
		actualObject, _ := json.MarshalIndent(actualResult.ContainerDefinitions, "", "  ")
		actualJSON := string(actualObject[:])
		expectedObject, _ := json.MarshalIndent(expectedResult, "", "  ")
		expectedJSON := string(expectedObject[:])
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedJSON, actualJSON, true)
		t.Fatal(dmp.DiffPrettyText(diffs))
	}
}

func TestAlterLogConfigurationsExistingSecret(t *testing.T) {
	testObject := ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Name: aws.String("container1"),
			},
			{
				Name: aws.String("container2"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver2"),
					SecretOptions: []*ecs.Secret{
						{
							Name:      aws.String("driver2Option1"),
							ValueFrom: aws.String("driver2Value1"),
						},
						{
							Name:      aws.String("driver2Option2"),
							ValueFrom: aws.String("driver2Value2"),
						},
					},
				},
			},
			{
				Name: aws.String("container3"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver3"),
					SecretOptions: []*ecs.Secret{
						{
							Name:      aws.String("driver3KeepOption1"),
							ValueFrom: aws.String("driver3Value1"),
						},
						{
							Name:      aws.String("driver3KeepOption2"),
							ValueFrom: aws.String("driver3Value2"),
						},
						{
							Name:      aws.String("driver3RemoveOption1"),
							ValueFrom: aws.String("driver3Value3"),
						},
					},
				},
			},
			{
				Name: aws.String("container4"),
			},
			{
				Name: aws.String("container5"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver5"),
					SecretOptions: []*ecs.Secret{
						{
							Name:      aws.String("driver5Option1"),
							ValueFrom: aws.String("driver5Value1"),
						},
						{
							Name:      aws.String("driver5Option2"),
							ValueFrom: aws.String("driver5Value2"),
						},
					},
				},
			},
			{
				Name: aws.String("container6"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver6"),
					SecretOptions: []*ecs.Secret{
						{
							Name:      aws.String("driver6Option1"),
							ValueFrom: aws.String("driver6Value1"),
						},
						{
							Name:      aws.String("driver6Option2"),
							ValueFrom: aws.String("driver6Value2"),
						},
						{
							Name:      aws.String("driver6UpdateOption1"),
							ValueFrom: aws.String("driver6OldValue1"),
						},
					},
				},
			},
			{
				Name: aws.String("container7"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("oldDriver7"),
					SecretOptions: []*ecs.Secret{
						{
							Name:      aws.String("driver7KeepOption1"),
							ValueFrom: aws.String("driver7Value1"),
						},
						{
							Name:      aws.String("driver7KeepOption2"),
							ValueFrom: aws.String("driver7Value2"),
						},
					},
				},
			},
			{
				Name: aws.String("container8"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver8"),
					SecretOptions: []*ecs.Secret{
						{
							Name:      aws.String("driver8Option1"),
							ValueFrom: aws.String("driver8Value1"),
						},
						{
							Name:      aws.String("driver8Option2"),
							ValueFrom: aws.String("driver8Value2"),
						},
					},
				},
			},
			{
				Name: aws.String("container9"),
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: aws.String("driver9"),
					SecretOptions: []*ecs.Secret{
						{
							Name:      aws.String("driver9Option1"),
							ValueFrom: aws.String("driver9Value1"),
						},
						{
							Name:      aws.String("driver9Option2"),
							ValueFrom: aws.String("driver9Value2"),
						},
					},
				},
			},
		},
	}
	actualResult := alterLogConfigurations(testObject, nil, map[string]map[string]map[string]string{
		"container3": {
			"driver3": {
				"driver3RemoveOption1": "",
			},
		},
		"container4": {
			"driver4": {
				"driver4Option1": "driver4Value1",
				"driver4Option2": "driver4Value2",
			},
		},
		"container5": {
			"driver5": {
				"driver5Option3": "driver5Value3",
			},
		},
		"container6": {
			"driver6": {
				"driver6UpdateOption1": "driver6NewValue1",
			},
		},
		"container7": {
			"oldDriver7": {
				"": "",
			},
			"newDriver7": {
				"newDriver7NewOption1": "newDriver7Value1",
			},
		},
		"container8": {
			"newDriver8": {
				"newDriver8Option1": "newDriver8Value1",
			},
		},
		"container9": {
			"driver9": {
				"": "",
			},
		},
	})
	expectedResult := []*ecs.ContainerDefinition{
		{
			Name: aws.String("container1"),
		},
		{
			Name: aws.String("container2"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver2"),
				SecretOptions: []*ecs.Secret{
					{
						Name:      aws.String("driver2Option1"),
						ValueFrom: aws.String("driver2Value1"),
					},
					{
						Name:      aws.String("driver2Option2"),
						ValueFrom: aws.String("driver2Value2"),
					},
				},
			},
		},
		{
			Name: aws.String("container3"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver3"),
				SecretOptions: []*ecs.Secret{
					{
						Name:      aws.String("driver3KeepOption1"),
						ValueFrom: aws.String("driver3Value1"),
					},
					{
						Name:      aws.String("driver3KeepOption2"),
						ValueFrom: aws.String("driver3Value2"),
					},
				},
			},
		},
		{
			Name: aws.String("container4"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver4"),
				SecretOptions: []*ecs.Secret{
					{
						Name:      aws.String("driver4Option1"),
						ValueFrom: aws.String("driver4Value1"),
					},
					{
						Name:      aws.String("driver4Option2"),
						ValueFrom: aws.String("driver4Value2"),
					},
				},
			},
		},
		{
			Name: aws.String("container5"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver5"),
				SecretOptions: []*ecs.Secret{
					{
						Name:      aws.String("driver5Option1"),
						ValueFrom: aws.String("driver5Value1"),
					},
					{
						Name:      aws.String("driver5Option2"),
						ValueFrom: aws.String("driver5Value2"),
					},
					{
						Name:      aws.String("driver5Option3"),
						ValueFrom: aws.String("driver5Value3"),
					},
				},
			},
		},
		{
			Name: aws.String("container6"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver6"),
				SecretOptions: []*ecs.Secret{
					{
						Name:      aws.String("driver6Option1"),
						ValueFrom: aws.String("driver6Value1"),
					},
					{
						Name:      aws.String("driver6Option2"),
						ValueFrom: aws.String("driver6Value2"),
					},
					{
						Name:      aws.String("driver6UpdateOption1"),
						ValueFrom: aws.String("driver6NewValue1"),
					},
				},
			},
		},
		{
			Name: aws.String("container7"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("newDriver7"),
				SecretOptions: []*ecs.Secret{
					{
						Name:      aws.String("driver7KeepOption1"),
						ValueFrom: aws.String("driver7Value1"),
					},
					{
						Name:      aws.String("driver7KeepOption2"),
						ValueFrom: aws.String("driver7Value2"),
					},
					{
						Name:      aws.String("newDriver7NewOption1"),
						ValueFrom: aws.String("newDriver7Value1"),
					},
				},
			},
		},
		{
			Name: aws.String("container8"),
			LogConfiguration: &ecs.LogConfiguration{
				LogDriver: aws.String("driver8"),
				SecretOptions: []*ecs.Secret{
					{
						Name:      aws.String("driver8Option1"),
						ValueFrom: aws.String("driver8Value1"),
					},
					{
						Name:      aws.String("driver8Option2"),
						ValueFrom: aws.String("driver8Value2"),
					},
				},
			},
		},
		{
			Name: aws.String("container9"),
		},
	}
	if !reflect.DeepEqual(actualResult.ContainerDefinitions, expectedResult) {
		actualObject, _ := json.MarshalIndent(actualResult.ContainerDefinitions, "", "  ")
		actualJSON := string(actualObject[:])
		expectedObject, _ := json.MarshalIndent(expectedResult, "", "  ")
		expectedJSON := string(expectedObject[:])
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedJSON, actualJSON, true)
		t.Fatal(dmp.DiffPrettyText(diffs))
	}
}
