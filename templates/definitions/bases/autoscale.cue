package bases

#HorizontalPodAutoscaler: {
	parameter: {
		name:        string
		minReplicas: int
		maxReplicas: int
		cpuPercent:  int
	}

	output: {
		apiVersion: "autoscaling/v2"
		kind:       "HorizontalPodAutoscaler"
		metadata: name: parameter.name
		spec: {
			scaleTargetRef: {
				apiVersion: "apps/v1"
				kind:       "Deployment"
				name:       parameter.name
			}
			minReplicas: parameter.minReplicas
			maxReplicas: parameter.maxReplicas
			metrics: [{
				type: "Resource"
				resource: {
					name: "cpu"
					target: {
						type:               "Utilization"
						averageUtilization: parameter.cpuPercent
					}
				}
			}]
		}
	}
}