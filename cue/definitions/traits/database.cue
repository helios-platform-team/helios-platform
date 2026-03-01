package traits

import "strings"

// DatabaseTrait — provisions database connection resources for a Component.
// Renders a ConfigMap (connection metadata) and a Secret (credentials).
//
// Usage via the trait system:
//   traits: [{
//       type: "database"
//       properties: {
//           dbType: "postgres"
//           dbName: "my_custom_db"
//       }
//   }]

// Supported database engine types.
#DatabaseType: "postgres" | "mysql" | "mongodb" | "redis"

// Default ports per engine type.
_#defaultPorts: {
	postgres: 5432
	mysql:    3306
	mongodb:  27017
	redis:    6379
}

#DatabaseTrait: {
	parameter: {
		name!:   string & strings.MinRunes(1)
		dbType!: #DatabaseType
		dbName:  string | *""
		port:    int & >0 & <=65535 | *_#defaultPorts[dbType]
		version: string | *"latest"
		storage: string & =~"^[0-9]+(Mi|Gi|Ti)$" | *"1Gi"

		// If non-empty, the trait skips Secret generation and references
		// this existing secret instead.
		secretName: string | *""
	}

	_p: parameter

	// Resolve the effective database name: user-provided or default.
	let _effectiveDBName = [
		if _p.dbName != "" {_p.dbName},
		"\(_p.name)-db",
	][0]

	// Resolve the effective secret name.
	let _effectiveSecretName = [
		if _p.secretName != "" {_p.secretName},
		"\(_p.name)-db-secret",
	][0]

	outputs: {
		// --- ConfigMap: non-sensitive connection metadata ---
		configmap: {
			apiVersion: "v1"
			kind:       "ConfigMap"
			metadata: {
				name: "\(_p.name)-db-config"
				labels: {
					app:                    _p.name
					"helios.io/managed-by": "operator"
					"helios.io/trait":      "database"
				}
			}
			data: {
				DB_TYPE:    _p.dbType
				DB_HOST:    "\(_p.name)-db"
				DB_PORT:    "\(_p.port)"
				DB_NAME:    _effectiveDBName
				DB_VERSION: _p.version
				DB_STORAGE: _p.storage
			}
		}

		// --- Secret: credentials (only if no external secret is referenced) ---
		if _p.secretName == "" {
			secret: {
				apiVersion: "v1"
				kind:       "Secret"
				metadata: {
					name: _effectiveSecretName
					labels: {
						app:                    _p.name
						"helios.io/managed-by": "operator"
						"helios.io/trait":      "database"
					}
				}
				type: "Opaque"
				stringData: {
					DB_USER:     "admin"
					DB_PASSWORD: "changeme"
				}
			}
		}
	}
}
