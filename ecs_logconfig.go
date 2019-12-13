package awsecs

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func alterLogConfigurationLogDriverOptions(copy ecs.LogConfiguration, overrides map[string]map[string]string) ecs.LogConfiguration {
	knockOutDriver := ""
	for logDriver, overrides := range overrides {
		if copy.LogDriver != nil && *copy.LogDriver == logDriver {
			for optionName, optionValue := range overrides {
				if optionName == EnvKnockOutValue {
					knockOutDriver = logDriver
				}
				if copy.Options == nil {
					copy.Options = map[string]*string{}
				}
				copy.Options[optionName] = aws.String(optionValue)
				if optionValue == EnvKnockOutValue || optionName == EnvKnockOutValue {
					delete(copy.Options, optionName)
				}
			}
		}
	}
	if knockOutDriver != "" {
		delete(overrides, knockOutDriver)
		copy.LogDriver = nil
	}
	if copy.LogDriver == nil && len(overrides) == 1 {
		for logDriver, options := range overrides {
			copy.LogDriver = aws.String(logDriver)
			for optionName, optionValue := range options {
				if copy.Options == nil {
					copy.Options = map[string]*string{}
				}
				copy.Options[optionName] = aws.String(optionValue)
				if optionValue == EnvKnockOutValue || optionName == EnvKnockOutValue {
					delete(copy.Options, optionName)
				}
			}
		}
	}
	return copy
}

func alterLogConfigurationLogDriverSecrets(copy ecs.LogConfiguration, overrides map[string]map[string]string) ecs.LogConfiguration {
	knockOutDriver := ""
	for logDriver, overrides := range overrides {
		if copy.LogDriver != nil && *copy.LogDriver == logDriver {
			for optionName, optionValue := range overrides {
				if optionName == EnvKnockOutValue {
					knockOutDriver = logDriver
				}
				thisOptionChanged := false
				for _, secretOption := range copy.SecretOptions {
					if secretOption != nil && secretOption.Name != nil && *secretOption.Name == optionName {
						thisOptionChanged = true
						secretOption.ValueFrom = aws.String(optionValue)
					}
				}
				if !thisOptionChanged {
					copy.SecretOptions = append(copy.SecretOptions, &ecs.Secret{Name: aws.String(optionName), ValueFrom: aws.String(optionValue)})
				}
				var filteredSecretOptions []*ecs.Secret
				for _, secretOption := range copy.SecretOptions {
					if secretOption != nil && secretOption.Name != nil && *secretOption.Name == optionName {
						if optionValue != EnvKnockOutValue && optionName != EnvKnockOutValue {
							filteredSecretOptions = append(filteredSecretOptions, secretOption)
						}
					} else {
						filteredSecretOptions = append(filteredSecretOptions, secretOption)
					}
				}
				copy.SecretOptions = filteredSecretOptions
			}
		}
	}
	if knockOutDriver != "" {
		delete(overrides, knockOutDriver)
		copy.LogDriver = nil
	}
	if copy.LogDriver == nil && len(overrides) == 1 {
		for logDriver, options := range overrides {
			copy.LogDriver = aws.String(logDriver)
			for optionName, optionValue := range options {
				thisOptionChanged := false
				for _, secretOption := range copy.SecretOptions {
					if secretOption != nil && secretOption.Name != nil && *secretOption.Name == optionName {
						thisOptionChanged = true
					}
				}
				if !thisOptionChanged {
					copy.SecretOptions = append(copy.SecretOptions, &ecs.Secret{Name: aws.String(optionName), ValueFrom: aws.String(optionValue)})
				}
				var filteredSecretOptions []*ecs.Secret
				for _, secretOption := range copy.SecretOptions {
					if secretOption != nil && secretOption.Name != nil && *secretOption.Name == optionName {
						if optionValue != EnvKnockOutValue && optionName != EnvKnockOutValue {
							filteredSecretOptions = append(filteredSecretOptions, secretOption)
						}
					} else {
						filteredSecretOptions = append(filteredSecretOptions, secretOption)
					}
				}
				copy.SecretOptions = filteredSecretOptions
			}
		}
	}
	return copy
}

func alterLogConfigurations(copy ecs.RegisterTaskDefinitionInput, containersOptions map[string]map[string]map[string]string, containersSecrets map[string]map[string]map[string]string) ecs.RegisterTaskDefinitionInput {
	obj, err := json.Marshal(copy)
	if err != nil {
		panic(err)
	}
	copyClone := ecs.RegisterTaskDefinitionInput{}
	err = json.Unmarshal(obj, &copyClone)
	if err != nil {
		panic(err)
	}
	for _, containerDefinition := range copyClone.ContainerDefinitions {
		for containerName, containerOptions := range containersOptions {
			if containerDefinition.Name != nil && *containerDefinition.Name == containerName {
				var logConfiguration *ecs.LogConfiguration
				if containerDefinition.LogConfiguration != nil {
					logConfiguration = containerDefinition.LogConfiguration
				} else {
					logConfiguration = &ecs.LogConfiguration{}
				}
				*logConfiguration = alterLogConfigurationLogDriverOptions(*logConfiguration, containerOptions)
				if logConfiguration.LogDriver == nil || (logConfiguration.LogDriver != nil && *logConfiguration.LogDriver == "") {
					containerDefinition.LogConfiguration = nil
				} else {
					containerDefinition.LogConfiguration = logConfiguration
				}
			}
		}
		for containerName, containerOptions := range containersSecrets {
			if containerDefinition.Name != nil && *containerDefinition.Name == containerName {
				var logConfiguration *ecs.LogConfiguration
				if containerDefinition.LogConfiguration != nil {
					logConfiguration = containerDefinition.LogConfiguration
				} else {
					logConfiguration = &ecs.LogConfiguration{}
				}
				*logConfiguration = alterLogConfigurationLogDriverSecrets(*logConfiguration, containerOptions)
				if logConfiguration.LogDriver == nil || (logConfiguration.LogDriver != nil && *logConfiguration.LogDriver == "") {
					containerDefinition.LogConfiguration = nil
				} else {
					containerDefinition.LogConfiguration = logConfiguration
				}
			}
		}
	}
	return copyClone
}
