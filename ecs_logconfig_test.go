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
		Options: map[string]*string{},
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
			"myAddedSecretOption": "myAddedSecretOptionNewValue",
			optionToDelete:    EnvKnockOutValue,
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
		Options: map[string]*string{},
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
			"": "ignore",
			optionToDelete1:    EnvKnockOutValue,
		},
		"mynewdriver": {
			"myAddedSecretOption": "myAddedSecretOptionNewValue",
			optionToDelete2:    EnvKnockOutValue,
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
