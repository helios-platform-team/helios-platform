package database

import (
	"fmt"
	"net/url"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var requiredDatabaseSecretKeys = []string{"DB_USER", "DB_PASS", "DB_HOST"}

// formatPostgresURI constructs a PostgreSQL connection URI from components.
// It properly escapes the username and password for use in URLs.
// Format: postgres://username:password@host:port/dbname?sslmode=disable
func formatPostgresURI(username, password, host, dbName string, port int32) string {
	// Escape username and password for use in URL
	enscodedUser := url.QueryEscape(username)
	enscodedPassword := url.QueryEscape(password)

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		enscodedUser,
		enscodedPassword,
		host,
		port,
		dbName,
	)
}

// GenerateDatabaseSecret creates a Kubernetes Secret containing database credentials.
// Parameters: namespace, secretName, componentName, credentials, dbHost, dbName, port.
func GenerateDatabaseSecret(namespace, secretName, componentName string, creds *DatabaseCredentials, dbHost, dbName string, port int32) *corev1.Secret {
	// Compute the PostgreSQL connection URI
	pgrstURI := formatPostgresURI(creds.Username, creds.Password, dbHost, dbName, port)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                   componentName,
				"helios.io/managed-by":  "operator",
				"helios.io/secret-type": "database-credentials",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"DB_USER":      []byte(creds.Username),
			"DB_PASS":      []byte(creds.Password),
			"DB_HOST":      []byte(dbHost),
			"PGRST_DB_URI": []byte(pgrstURI),
		},
	}
}

// ValidateDatabaseSecret ensures an existing database credential Secret has all
// required keys and the expected DB_HOST value.
func ValidateDatabaseSecret(secret *corev1.Secret, expectedHost string) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}

	missingKeys := make([]string, 0, len(requiredDatabaseSecretKeys))
	for _, key := range requiredDatabaseSecretKeys {
		value, ok := secret.Data[key]
		if !ok || len(value) == 0 {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("missing required keys: %s", strings.Join(missingKeys, ", "))
	}

	if expectedHost != "" && string(secret.Data["DB_HOST"]) != expectedHost {
		return fmt.Errorf("DB_HOST mismatch: got %q, expected %q", string(secret.Data["DB_HOST"]), expectedHost)
	}

	return nil
}

// GenerateDatabaseStatefulSet creates a StatefulSet for a Postgres database instance.
func GenerateDatabaseStatefulSet(namespace, name, secretName, dbName, version, storage string, port int32) (*appsv1.StatefulSet, error) {
	storageQty, err := resource.ParseQuantity(storage)
	if err != nil {
		return nil, fmt.Errorf("invalid storage size format %q: %w", storage, err)
	}

	replicas := int32(1)
	labels := map[string]string{
		"app":                  name,
		"helios.io/managed-by": "operator",
		"helios.io/trait":      "database",
		"helios.io/db-type":    dbTypePostgres,
	}

	probeCommand := `pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" -p "$PGPORT"`

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  dbTypePostgres,
							Image: fmt.Sprintf("postgres:%s", version),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: port,
									Name:          dbTypePostgres,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "POSTGRES_DB",
									Value: dbName,
								},
								{
									Name: "POSTGRES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: secretName,
											},
											Key: "DB_USER",
										},
									},
								},
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: secretName,
											},
											Key: "DB_PASS",
										},
									},
								},
								{
									Name:  "PGDATA",
									Value: PostgresDataPath,
								},
								{
									Name:  "POSTGRES_INITDB_ARGS",
									Value: "--encoding=UTF-8 --lc-collate=C --lc-ctype=C",
								},
								{
									Name:  "PGPORT",
									Value: fmt.Sprintf("%d", port),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: PostgresDataPath,
									SubPath:   PostgresDataSubPath,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"sh", "-c", probeCommand},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"sh", "-c", probeCommand},
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "data",
						Labels: labels,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: storageQty,
							},
						},
					},
				},
			},
		},
	}, nil
}

// GenerateDatabaseService creates a headless Service for a database StatefulSet.
func GenerateDatabaseService(namespace, name string, port int32) *corev1.Service {
	labels := map[string]string{
		"app":                  name,
		"helios.io/managed-by": "operator",
		"helios.io/trait":      "database",
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  map[string]string{"app": name},
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: intstr.FromInt32(port),
					Name:       "db",
				},
			},
		},
	}
}
