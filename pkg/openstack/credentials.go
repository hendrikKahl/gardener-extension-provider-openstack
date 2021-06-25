// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openstack

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Credentials contains the necessary OpenStack credential information.
type Credentials struct {
	DomainName string
	TenantName string

	// either authenticate with username/password credentials
	Username string
	Password string

	// or application credentials
	ApplicationCredentialID     string
	ApplicationCredentialSecret string

	AuthURL string
}

// GetCredentials computes for a given context and infrastructure the corresponding credentials object.
func GetCredentials(ctx context.Context, c client.Client, secretRef corev1.SecretReference) (*Credentials, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, c, &secretRef)
	if err != nil {
		return nil, err
	}
	return ExtractCredentials(secret)
}

// ExtractCredentials generates a credentials object for a given provider secret.
func ExtractCredentials(secret *corev1.Secret) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}
	domainName, err := getRequired(secret, DomainName)
	if err != nil {
		return nil, err
	}
	tenantName, err := getRequired(secret, TenantName)
	if err != nil {
		return nil, err
	}
	userName := getOptional(secret, UserName)
	applicationCredentialID := getOptional(secret, ApplicationCredentialID)
	authURL := getOptional(secret, AuthURL)

	var password, applicationCredentialSecret string
	if userName != "" {
		if applicationCredentialID != "" {
			return nil, fmt.Errorf("cannot specify both '%s' and '%s' in secret %s/%s", UserName, ApplicationCredentialID, secret.Namespace, secret.Name)
		}
		password, err = getRequired(secret, Password)
		if err != nil {
			return nil, err
		}
	} else {
		if applicationCredentialID == "" {
			return nil, fmt.Errorf("must either specify '%s' or '%s' in secret %s/%s", UserName, ApplicationCredentialID, secret.Namespace, secret.Name)
		}
		applicationCredentialSecret, err = getRequired(secret, ApplicationCredentialSecret)
		if err != nil {
			return nil, err
		}
	}

	return &Credentials{
		DomainName:                  domainName,
		TenantName:                  tenantName,
		Username:                    userName,
		Password:                    password,
		ApplicationCredentialID:     applicationCredentialID,
		ApplicationCredentialSecret: applicationCredentialSecret,
		AuthURL:                     string(authURL),
	}, nil
}

// getOptional returns optional value for a corresponding key or empty string
func getOptional(secret *corev1.Secret, key string) string {
	if value, ok := secret.Data[key]; ok {
		return string(value)
	}
	return ""
}

// getRequired checks if the provided map has a valid value for a corresponding key.
func getRequired(secret *corev1.Secret, key string) (string, error) {
	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("missing %q data key in secret %s/%s", key, secret.Namespace, secret.Name)
	}
	if len(value) == 0 {
		return "", fmt.Errorf("key %q in secret %s/%s cannot be empty", key, secret.Namespace, secret.Name)
	}
	return string(value), nil
}
