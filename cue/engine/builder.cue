package engine

import (
	"list"
	"helios.io/cue/definitions/schema"
	"helios.io/cue/definitions/components"
	"helios.io/cue/definitions/traits"
)

// Input: Application Model từ Operator
input: schema.#HeliosApp

#DatabaseTraitRef: {
	type: "database"
	properties: {
		dbType: string | *"postgres"
		dbName: string | *""
		port:   int | *5432
	}
}

// =============================================================================
// REGISTRIES - Chỉ làm lookup, KHÔNG chứa logic
// =============================================================================

// Component Registry - mapping type name → component definition
#ComponentRegistry: {
	"web-service": components.#WebService
	// Thêm component mới chỉ cần thêm dòng ở đây
	// "java-service": components.#JavaService
	// "dotnet-service": components.#DotNetService
}

// Trait Registry - mapping type name → trait definition
#TraitRegistry: {
	"service":                   traits.#ServiceTrait
	"ingress":                   traits.#IngressTrait
	"database":                  traits.#DatabaseTrait
	"external-secret-reference": traits.#ExternalSecretReferenceTrait
	// Thêm trait mới chỉ cần thêm dòng ở đây
	// "autoscale": traits.#AutoscaleTrait
	// "servicemonitor": traits.#ServiceMonitorTrait
}

// =============================================================================
// GENERIC RENDERING - Không có if/else theo type, không đọc field cụ thể
// =============================================================================

// Generic component rendering - ONE loop for ALL component types
componentsRendered: {
	for comp in input.app.components {
		let Def = #ComponentRegistry[comp.type]

		// external-secret-reference: merged into web-service Deployment envFrom (Git-visible wiring).
		let envFromSecretNames = [
			if comp.traits != _|_ for trait in comp.traits if trait.type == "external-secret-reference" {
				trait.properties.secretName
			},
		]

		// Find database trait if any
		let dbTraits = [...#DatabaseTraitRef] & [
			if comp.traits != _|_
			for t in comp.traits if t.type == "database" {
				#DatabaseTraitRef & t
			}
		]
		let hasDb = len(dbTraits) > 0
		
		let baseEnv = [
			if comp.properties.env != _|_ { comp.properties.env },
			[]
		][0]

		let cleanProperties = {
			for k, v in comp.properties if k != "env" {
				"\(k)": v
			}
		}

		let dbEnvList = [
			for dbT in dbTraits {
				let dbType = dbT.properties.dbType
				let dbName = [
					if dbT.properties.dbName != _|_ && dbT.properties.dbName != "" { dbT.properties.dbName },
					"\(comp.name)-db"
				][0]
				let dbSecretName = "\(comp.name)-db-secret"
				let dbPort = [
					if dbT.properties.port != _|_ { dbT.properties.port },
					if dbType == "postgres" { 5432 },
					if dbType == "mysql" { 3306 },
					if dbType == "mongodb" { 27017 },
					if dbType == "redis" { 6379 }
				][0]
				
				[
					{name: "DB_TYPE", value: dbType},
					{name: "DB_HOST", valueFrom: secretKeyRef: {name: dbSecretName, key: "DB_HOST"}},
					{name: "DB_PORT", value: "\(dbPort)"},
					{name: "DB_NAME", value: dbName},
					{name: "DB_USER", valueFrom: secretKeyRef: {name: dbSecretName, key: "DB_USER"}},
					{name: "DB_PASS", valueFrom: secretKeyRef: {name: dbSecretName, key: "DB_PASS"}},
					{name: "PGRST_DB_URI", valueFrom: secretKeyRef: {name: dbSecretName, key: "PGRST_DB_URI"}},
					{name: "DATABASE_URL", value: "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)"}
				]
			}
		]
		let dbEnv = [
			if len(dbEnvList) > 0 { dbEnvList[0] },
			[]
		][0]

		let initContsList = [
			for dbT in dbTraits {
				let dbType = dbT.properties.dbType
				let dbPort = [
					if dbT.properties.port != _|_ { dbT.properties.port },
					if dbType == "postgres" { 5432 },
					if dbType == "mysql" { 3306 },
					if dbType == "mongodb" { 27017 },
					if dbType == "redis" { 6379 }
				][0]
				[{
					name: "wait-for-db"
					image: "busybox:1.37"
					command: ["sh", "-c", "until nc -z $DB_HOST $DB_PORT; do echo 'waiting for database...'; sleep 2; done"]
					env: [
						{
							name: "DB_HOST"
							value: "\(comp.name)-db"
						},
						{
							name: "DB_PORT"
							value: "\(dbPort)"
						}
					]
				}]
			}
		]
		let initConts = [
			if len(initContsList) > 0 { initContsList[0] },
			[]
		][0]

		"\(comp.name)": (Def & {
			parameter: cleanProperties & {
				name: comp.name
				env: list.Concat([baseEnv, dbEnv])
				if hasDb {
					initContainers: initConts
				}
				if len(envFromSecretNames) > 0 {
					envFromSecrets: envFromSecretNames
				}
			}
		}).outputs
	}
}

// Generic trait rendering - ONE loop for ALL trait types
traitsRendered: {
	for comp in input.app.components
	if comp.traits != _|_
	for trait in comp.traits {
		let TraitDef = #TraitRegistry[trait.type]
		"\(comp.name)-\(trait.type)": (TraitDef & {
			parameter: trait.properties & {
				name: comp.name
			}
		}).outputs
	}
}

// =============================================================================
// OUTPUT AGGREGATION
// =============================================================================

// Final Kubernetes Objects - flatten all rendered outputs
kubernetesObjects: [
	// Collect all component outputs
	for _, compOut in componentsRendered
	for _, res in compOut {
		res
	},
	// Collect all trait outputs
	for _, traitOut in traitsRendered
	for _, res in traitOut {
		res
	},
]
