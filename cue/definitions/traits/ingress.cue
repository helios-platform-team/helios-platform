package traits

// IngressTrait - Tạo Ingress cho Component

#IngressTrait: {
	parameter: {
		name:      string
		host:      string
		path:      string | *"/"
		port:      int
		className: string | *"nginx"
	}

	_p: parameter

	outputs: {
		ingress: {
			apiVersion: "networking.k8s.io/v1"
			kind:       "Ingress"
			metadata: {
				name: "\(_p.name)-ingress"
				labels: {
					app:                    _p.name
					"helios.io/managed-by": "operator"
				}
				annotations: "kubernetes.io/ingress.class": _p.className
			}
			spec: {
				ingressClassName: _p.className
				rules: [{
					host: _p.host
					http: paths: [{
						path:     _p.path
						pathType: "Prefix"
						backend: service: {
							name: _p.name
							port: number: _p.port
						}
					}]
				}]
			}
		}
	}
}
