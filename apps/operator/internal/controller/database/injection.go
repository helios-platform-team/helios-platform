package database

import (
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const dbPortEnvName = "DB_PORT"

// databaseSecretEnvVarNames lists env vars resolved from Secret keys.
var databaseSecretEnvVarNames = []string{"DB_HOST", "DB_USER", "DB_PASS"}

// InjectDatabaseEnvVars patches a Deployment's first container to include
// DB_HOST, DB_USER, DB_PASS env vars referencing the given K8s Secret.
// The function is idempotent.
func InjectDatabaseEnvVars(deploy *appsv1.Deployment, secretName string) bool {
	changed, _ := InjectDatabaseEnvVarsForContainer(deploy, secretName, "", DefaultPostgresPort)
	return changed
}

// InjectDatabaseEnvVarsForContainer patches a Deployment container to include
// DB_HOST, DB_USER, DB_PASS env vars referencing the given K8s Secret, plus a
// literal DB_PORT value.
// If preferredContainerName is not found, it falls back to the first container.
// Returns (changed, exactMatch).
func InjectDatabaseEnvVarsForContainer(deploy *appsv1.Deployment, secretName, preferredContainerName string, dbPort int32) (bool, bool) {
	if len(deploy.Spec.Template.Spec.Containers) == 0 {
		return false, false
	}

	containerIndex, exactMatch := selectTargetContainerIndex(deploy.Spec.Template.Spec.Containers, preferredContainerName)
	if containerIndex < 0 {
		return false, false
	}

	container := &deploy.Spec.Template.Spec.Containers[containerIndex]

	changed := false
	for _, envName := range databaseSecretEnvVarNames {
		expectedRef := &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: envName,
			},
		}

		found := false
		for i := range container.Env {
			if container.Env[i].Name != envName {
				continue
			}
			found = true
			if container.Env[i].ValueFrom != nil &&
				container.Env[i].ValueFrom.SecretKeyRef != nil &&
				container.Env[i].ValueFrom.SecretKeyRef.Name == secretName &&
				container.Env[i].ValueFrom.SecretKeyRef.Key == envName {
				break
			}
			container.Env[i].Value = ""
			container.Env[i].ValueFrom = expectedRef
			changed = true
			break
		}

		if !found {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:      envName,
				ValueFrom: expectedRef,
			})
			changed = true
		}
	}

	portValue := strconv.FormatInt(int64(dbPort), 10)
	foundDBPort := false
	for i := range container.Env {
		if container.Env[i].Name != dbPortEnvName {
			continue
		}
		foundDBPort = true
		if container.Env[i].Value == portValue && container.Env[i].ValueFrom == nil {
			break
		}
		container.Env[i].Value = portValue
		container.Env[i].ValueFrom = nil
		changed = true
		break
	}

	if !foundDBPort {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  dbPortEnvName,
			Value: portValue,
		})
		changed = true
	}

	return changed, exactMatch
}

func selectTargetContainerIndex(containers []corev1.Container, preferredContainerName string) (int, bool) {
	if len(containers) == 0 {
		return -1, false
	}

	if preferredContainerName == "" {
		return 0, false
	}

	for i := range containers {
		if containers[i].Name == preferredContainerName {
			return i, true
		}
	}

	return 0, false
}
