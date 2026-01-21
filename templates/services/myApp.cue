package services

import "helios.io/templates/definitions"

myApp: definitions.#Workload & {
    name:  "helios-api"
    image: "nginx:latest"
    replicas: 2
    port: 80
}