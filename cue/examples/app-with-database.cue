package examples

import "helios.io/cue/definitions/schema"

// Example: Web application with a PostgreSQL database trait.
// Validates acceptance criteria:
//   - app.database.type: "postgres"
//   - app.database.name: "my_custom_db"

appWithDatabase: schema.#HeliosApp & {
	app: {
		name:        "my-app"
		namespace:   "default"
		owner:       "dev-team"
		description: "Web application backed by a PostgreSQL database"

		components: [{
			name: "api-server"
			type: "web-service"
			properties: {
				image:    "myregistry/api:v2.0.0"
				port:     3000
				replicas: 2
			}
			traits: [
				{
					type: "service"
					properties: port: 3000
				},
				{
					type: "database"
					properties: {
						dbType:     "postgres"
						dbName:     "my_custom_db"
						version:    "16"
						dbUser:     "app_user"
						dbPassword: "<your-db-password>"
					}
				},
			]
		}]
	}
}
