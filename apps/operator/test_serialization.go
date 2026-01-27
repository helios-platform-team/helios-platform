package main

import (
	"encoding/json"
	"fmt"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func main() {
	// Mock App
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:   "http://git",
			ImageRepo: "docker/img",
		},
	}

	// Test 1: Generate EventListener (Deep nested []any)
	el, _ := controller.GenerateEventListener("el", "ns", "trig", "bind", "def", "tmpl", "sec")
	printYAML("EventListener", el.Object)

	// Test 2: Generate Ingress ([]any inside map)
	// Mock Domain to trigger ingress gen
	app.Spec.WebhookDomain = "example.com"
	ing, _ := controller.GenerateIngress(app, "el")
	printYAML("Ingress", ing.Object)
}

func printYAML(name string, obj interface{}) {
	fmt.Printf("--- %s ---\n", name)
	// Marshal to JSON first as k8s does
	j, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	// Convert to YAML
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(y))
}
