import { LoggerService } from '@backstage/backend-plugin-api';
import {
  HELIOS_CRD,
  HeliosAppResource,
  HeliosComponent,
  HeliosTrait,
} from './K8sSecretService';
import * as k8s from '@kubernetes/client-node';

const externalSecretReferenceTraitType = 'external-secret-reference';

export class HeliosAppManager {
  constructor(
    private readonly logger: LoggerService,
    private readonly k8sCustomApi: k8s.CustomObjectsApi,
  ) {}

  /**
   * Fetches the live CR from the cluster strictly to extract GitOps routing configurations.
   */
  async getGitOpsConfigFromCluster(serviceName: string, namespace: string) {
    try {
      const heliosApp = (await this.k8sCustomApi.getNamespacedCustomObject({
        ...HELIOS_CRD,
        namespace,
        name: serviceName,
      })) as HeliosAppResource;

      const gitopsRepo = heliosApp.spec?.gitopsRepo;
      const gitopsPath = heliosApp.spec?.gitopsPath;
      const gitopsSecretRef = heliosApp.spec?.gitopsSecretRef;

      if (!gitopsRepo || !gitopsPath) {
        throw new Error(
          `GitOps configurations (gitopsRepo/gitopsPath) are missing on live CR ${serviceName}`,
        );
      }

      return { gitopsRepo, gitopsPath, gitopsSecretRef };
    } catch (err: any) {
      this.logger.error(
        `Failed to look up GitOps coordinates from cluster for ${serviceName}: ${err.message}`,
      );
      throw err;
    }
  }

  applyExternalSecretTrait(
    heliosApp: HeliosAppResource,
    serviceName: string,
    secretName: string,
  ): HeliosAppResource {
    const components = heliosApp.spec?.components ?? [];
    const targetIndex = components.findIndex(c => c?.name === serviceName);

    if (targetIndex < 0) {
      throw new Error(
        `Component ${serviceName} not found in the GitOps manifest structure.`,
      );
    }

    const updatedComponents = [...components];
    updatedComponents[targetIndex] = this.#upsertTraitOnComponent(
      components[targetIndex],
      secretName,
    );

    return {
      ...heliosApp,
      spec: { ...heliosApp.spec, components: updatedComponents },
    };
  }

  removeExternalSecretTrait(
    heliosApp: HeliosAppResource,
    serviceName: string,
    secretName: string,
  ): HeliosAppResource {
    const components = heliosApp.spec?.components ?? [];
    const targetIndex = components.findIndex(c => c?.name === serviceName);

    if (targetIndex < 0) {
      this.logger.warn(
        `Component ${serviceName} not found during trait removal execution.`,
      );
      return heliosApp;
    }

    const updatedComponents = [...components];
    updatedComponents[targetIndex] = this.#removeTraitFromComponent(
      components[targetIndex],
      secretName,
    );

    return {
      ...heliosApp,
      spec: { ...heliosApp.spec, components: updatedComponents },
    };
  }

  #upsertTraitOnComponent(
    component: HeliosComponent,
    secretName: string,
  ): HeliosComponent {
    const traits = Array.isArray(component.traits) ? [...component.traits] : [];
    const externalSecretTrait: HeliosTrait = {
      type: externalSecretReferenceTraitType,
      properties: { secretName },
    };

    const hasSameSecretTrait = traits.some(
      t =>
        t?.type === externalSecretReferenceTraitType &&
        t?.properties?.secretName === secretName,
    );

    if (!hasSameSecretTrait) {
      traits.push(externalSecretTrait);
    }

    return { ...component, traits };
  }

  #removeTraitFromComponent(
    component: HeliosComponent,
    secretName: string,
  ): HeliosComponent {
    const traits = Array.isArray(component.traits) ? [...component.traits] : [];
    const filteredTraits = traits.filter(
      t =>
        !(
          t?.type === externalSecretReferenceTraitType &&
          t?.properties?.secretName === secretName
        ),
    );

    return { ...component, traits: filteredTraits };
  }
}
