package database

import (
	"encoding/json"
	"strings"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultPasswordLength = 32
	DefaultUsernameLength = 16

	PasswordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	UsernameCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

	DatabaseTraitType      = "database"
	DefaultPostgresVersion = "18.3"
	DefaultPostgresPort    = 5432
	DefaultDatabaseStorage = "1Gi"
	PostgresDataPath       = "/var/lib/postgresql/data"
	PostgresDataSubPath    = "pgdata"
)

// DatabaseCredentials holds generated database credentials.
type DatabaseCredentials struct {
	Username string
	Password string
}

// DatabaseTraitProperties represents the properties of a database trait.
type DatabaseTraitProperties struct {
	DBType  string `json:"dbType"`
	DBName  string `json:"dbName"`
	Port    int    `json:"port"`
	Version string `json:"version"`
	Storage string `json:"storage"`
}

// DatabaseTrait pairs a component name with its database trait properties.
type DatabaseTrait struct {
	ComponentName string
	Properties    DatabaseTraitProperties
}

// ExtractDatabaseTraits extracts all database traits from HeliosApp components.
func ExtractDatabaseTraits(app *appv1alpha1.HeliosApp) []DatabaseTrait {
	log := logf.Log.WithName("database-traits")

	var dbTraits []DatabaseTrait

	for _, component := range app.Spec.Components {
		for _, trait := range component.Traits {
			if strings.ToLower(trait.Type) == DatabaseTraitType {
				var props DatabaseTraitProperties
				if trait.Properties != nil && trait.Properties.Raw != nil {
					if err := json.Unmarshal(trait.Properties.Raw, &props); err != nil {
						log.Error(err, "Failed to parse database trait properties, skipping trait",
							"component", component.Name,
							"traitType", trait.Type,
							"rawPreview", truncateForLog(trait.Properties.Raw, 200))
						continue
					}
				}
				dbTraits = append(dbTraits, DatabaseTrait{
					ComponentName: component.Name,
					Properties:    props,
				})
			}
		}
	}

	return dbTraits
}

// GetDatabaseSecretName returns the conventional secret name for a component.
func GetDatabaseSecretName(componentName string) string {
	return componentName + "-db-secret"
}

// GetDatabaseHost returns the conventional database host for a component.
func GetDatabaseHost(componentName string) string {
	return componentName + "-db"
}

func truncateForLog(raw []byte, maxLen int) string {
	if maxLen <= 0 || len(raw) <= maxLen {
		return string(raw)
	}
	return string(raw[:maxLen]) + "..."
}
