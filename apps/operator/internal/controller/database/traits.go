package database

import (
	"encoding/json"
	"fmt"
	"strings"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultPasswordLength = 32
	DefaultUsernameLength = 16

	PasswordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	UsernameCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

	TraitType              = "database"
	DefaultPostgresVersion = "18.4"
	DefaultPostgresPort    = 5432
	DefaultDatabaseStorage = "1Gi"
	PostgresDataPath       = "/var/lib/postgresql/data"
	PostgresDataSubPath    = "pgdata"
)

// Credentials holds generated database credentials.
type Credentials struct {
	Username string
	Password string
}

// TraitProperties represents the properties of a database trait.
type TraitProperties struct {
	DBType  string `json:"dbType"`
	DBName  string `json:"dbName"`
	Port    int    `json:"port"`
	Version string `json:"version"`
	Storage string `json:"storage"`
}

// Trait pairs a component name with its database trait properties.
type Trait struct {
	ComponentName string
	Properties    TraitProperties
}

// ExtractDatabaseTraits extracts all database traits from HeliosApp components.
func ExtractDatabaseTraits(app *appv1alpha1.HeliosApp) []Trait {
	log := logf.Log.WithName("database-traits")

	var dbTraits []Trait

	for _, component := range app.Spec.Components {
		for _, trait := range component.Traits {
			if strings.ToLower(trait.Type) == TraitType {
				var props TraitProperties
				if trait.Properties != nil && trait.Properties.Raw != nil {
					if err := json.Unmarshal(trait.Properties.Raw, &props); err != nil {
						log.Error(err, "Failed to parse database trait properties, skipping trait",
							"component", component.Name,
							"traitType", trait.Type,
							"rawPreview", truncateForLog(trait.Properties.Raw, 200))
						continue
					}
				}
				dbTraits = append(dbTraits, Trait{
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

// EffectiveDatabaseName returns the logical Postgres database name for a trait,
// matching ReconcileInstances (POSTGRES_DB / connection string).
func EffectiveDatabaseName(tr Trait) string {
	if tr.Properties.DBName != "" {
		return tr.Properties.DBName
	}
	return fmt.Sprintf("%s-db", tr.ComponentName)
}

func truncateForLog(raw []byte, maxLen int) string {
	if maxLen <= 0 || len(raw) <= maxLen {
		return string(raw)
	}
	return string(raw[:maxLen]) + "..."
}
