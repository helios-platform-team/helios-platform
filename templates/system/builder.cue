package main

import (
	"helios.io/templates/schema"
	"helios.io/templates/definitions/components"
	"helios.io/templates/definitions/traits"
)

// Input application must satisfy the Helios schema
app: schema.#HeliosApp

// Component type registry - maps component types to their definitions
#ComponentRegistry: {
	"web-service": components.#WebService
}

// Trait type registry - maps trait types to their definitions
#TraitRegistry: {
	"service":   traits.#ServiceTrait
	"ingress":   traits.#IngressTrait
	"autoscale": traits.#AutoscaleTrait
}

// Render components
_components: {
	for comp in app.components
	let compName = comp.name
	let compImage = comp.properties.image
	let compReplicas = comp.properties.replicas
	let compPort = comp.properties.port {
		"\(compName)": {
			"web-service": components.#WebService & {
				parameter: {
					name:     compName
					image:    compImage
					replicas: compReplicas
					port:     compPort
				}
			}
		}[comp.type]
	}
}

// Render traits
_traits: {
	for comp in app.components
	let compName = comp.name {
		"\(compName)": {
			for trait in comp.traits
			let traitProps = trait.properties {
				"\(trait.type)": {
					"service": traits.#ServiceTrait & {
						parameter: {
                            traitProps
                            componentName: compName 
                        }
					}
					"ingress": traits.#IngressTrait & {
						parameter: traitProps
					}
					"autoscale": traits.#AutoscaleTrait & {
						parameter: traitProps
					}
				}[trait.type]
			}
		}
	}
}

// Main execution: generate all Kubernetes objects
kubernetesObjects: [
	for compName, compInstance in _components
	for _, outputValue in compInstance.outputs {
		outputValue.output
	},
	for compName, traitSet in _traits
	for traitName, traitInstance in traitSet
	for _, traitOutputValue in traitInstance.outputs {
		traitOutputValue.output
	},
]

// Optional: Export individual objects by type for easier inspection
deployments: [
	for obj in kubernetesObjects
	if obj.kind == "Deployment" {
		obj
	},
]

services: [
	for obj in kubernetesObjects
	if obj.kind == "Service" {
		obj
	},
]

ingresses: [
	for obj in kubernetesObjects
	if obj.kind == "Ingress" {
		obj
	},
]

horizontalPodAutoscalers: [
	for obj in kubernetesObjects
	if obj.kind == "HorizontalPodAutoscaler" {
		obj
	},
]