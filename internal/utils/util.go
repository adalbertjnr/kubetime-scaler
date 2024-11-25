package utils

import "os"

const namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func GetNamespace() (string, error) {
	namespace, err := os.ReadFile(namespacePath)
	if err != nil {
		return "", err
	}
	return string(namespace), nil
}
