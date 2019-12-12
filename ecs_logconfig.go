package awsecs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func alterLogConfigurationLogDriverOptions(copy ecs.LogConfiguration, overrides map[string]map[string]string) ecs.LogConfiguration {
	optionChanged := false
	knockOutDriver := ""
	for logDriver, overrides := range overrides {
		if copy.LogDriver != nil && *copy.LogDriver == logDriver {
			for optionName, optionValue := range overrides {
				if optionName == EnvKnockOutValue {
					knockOutDriver = logDriver
				}
				optionChanged = true
				copy.Options[optionName] = aws.String(optionValue)
				if optionValue == EnvKnockOutValue || optionName == EnvKnockOutValue {
					delete(copy.Options, optionName)
				}
			}
		}
	}
	if knockOutDriver != "" {
		optionChanged = false
		delete(overrides, knockOutDriver)
		copy.LogDriver = nil
	}
	if !optionChanged && len(overrides) == 1 {
		for logDriver, options := range overrides {
			copy.LogDriver = aws.String(logDriver)
			for optionName, optionValue := range options {
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
	optionChanged := false
	knockOutDriver := ""
	for logDriver, overrides := range overrides {
		if copy.LogDriver != nil && *copy.LogDriver == logDriver {
			for optionName, optionValue := range overrides {
				if optionName == EnvKnockOutValue {
					knockOutDriver = logDriver
				}
				optionChanged = true
				thisOptionChanged := false
				for _, secretOption := range copy.SecretOptions {
					if secretOption != nil && secretOption.Name != nil && *secretOption.Name == optionName {
						thisOptionChanged = true
						secretOption.ValueFrom = aws.String(optionValue)
					}
				}
				if ! thisOptionChanged {
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
		optionChanged = false
		delete(overrides, knockOutDriver)
		copy.LogDriver = nil
	}
	if !optionChanged && len(overrides) == 1 {
		for logDriver, options := range overrides {
			copy.LogDriver = aws.String(logDriver)
			for optionName, optionValue := range options {
				thisOptionChanged := false
				for _, secretOption := range copy.SecretOptions {
					if secretOption != nil && secretOption.Name != nil && *secretOption.Name == optionName {
						thisOptionChanged = true
					}
				}
				if ! thisOptionChanged {
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
