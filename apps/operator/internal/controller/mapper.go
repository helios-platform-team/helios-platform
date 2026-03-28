package controller

import (
	"encoding/json"
	"fmt"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
)

// mapCRDToModel converts HeliosApp CRD to CUE Application Model.
func mapCRDToModel(app *appv1alpha1.HeliosApp) (heliosCue.Application, error) {
	components := make([]heliosCue.Component, len(app.Spec.Components))

	for i, c := range app.Spec.Components {
		var props map[string]any
		if c.Properties != nil && c.Properties.Raw != nil {
			if err := json.Unmarshal(c.Properties.Raw, &props); err != nil {
				return heliosCue.Application{}, fmt.Errorf("failed to parse component properties: %w", err)
			}
		}

		traits := make([]heliosCue.Trait, len(c.Traits))
		for j, t := range c.Traits {
			var traitProps map[string]any
			if t.Properties != nil && t.Properties.Raw != nil {
				if err := json.Unmarshal(t.Properties.Raw, &traitProps); err != nil {
					return heliosCue.Application{}, fmt.Errorf("failed to parse trait properties: %w", err)
				}
			}
			traits[j] = heliosCue.Trait{
				Type:       t.Type,
				Properties: traitProps,
			}
		}

		components[i] = heliosCue.Component{
			Name:       c.Name,
			Type:       c.Type,
			Properties: props,
			Traits:     traits,
		}
	}

	return heliosCue.Application{
		App: heliosCue.AppSpec{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Owner:       app.Spec.Owner,
			Description: app.Spec.Description,
			Components:  components,
		},
	}, nil
}
