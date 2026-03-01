package engine

import (
	"helios.io/cue/definitions/schema"
	"helios.io/cue/definitions/components"
	"helios.io/cue/definitions/traits"
)

// Input: Application Model từ Operator
input: schema.#HeliosApp

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
	"service":  traits.#ServiceTrait
	"ingress":  traits.#IngressTrait
	"database": traits.#DatabaseTrait
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
		"\(comp.name)": (Def & {
			parameter: comp.properties & {
				name: comp.name
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
