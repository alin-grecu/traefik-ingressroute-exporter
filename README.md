# Traefik IngressRoute Prometheus Exporter

This application is designed to monitor the availability of tenants in a Kubernetes cluster using Prometheus metrics. It periodically fetches IngressRoute resources from different namespaces and updates the tenant_up metric based on the availability of each tenant.

## Prerequisites
Before running the application, make sure you have the following requirements met:

* **Kubernetes Cluster**: You need access to a running Kubernetes cluster.

* **Prometheus**: Prometheus must be installed and configured in your cluster to collect metrics.

* **Traefik**: The IngressRoute CRDs must be present.

## Installation
* Clone the repository

```bash
git clone https://github.com/alin-grecu/traefik-ingressroute-exporter.git
cd traefik-ingressroute-exporter
```
* Build the application

```bash
make build
```

* Start the application (example)
```bash
out/darwin/arm64/traefik-ingressroute-exporter start
```

## Configuration
The application automatically detects the Kubernetes cluster configuration. If you are running it outside the cluster, set the **KUBECONFIG** environment variable to the path of your kubeconfig file.

## Metrics
The application exposes the following Prometheus metric:

* **tenant_up**: A gauge that indicates the availability status of each tenant. The domain that is being checked along with the path are available though the **domain** label.

## Annotations
The scrape target for each tenant is determined based on specific annotations on the IngressRoute resources. Ensure the following annotations are defined for each IngressRoute resource you want to monitor:

* **traefik-ingressroute-exporter/domain**: The domain of the tenant.
* **traefik-ingressroute-exporter/path**: The path of the tenant.

## Metrics Scraping
The application periodically scrapes the IngressRoute resources from all namespaces in the Kubernetes cluster. It then makes HTTPS requests to the specified domains and paths to check the status of each tenant. The results are stored in the tenant_up metric.

## Monitoring
The application exposes the collected metrics on the **/metrics** endpoint. You can configure Prometheus to scrape these metrics for monitoring purposes.

## License
This application is licensed under the MIT License. See the LICENSE file for more details.

## Contribution
If you find any issues or have suggestions for improvements, feel free to open an issue or create a pull request on the GitHub repository.
