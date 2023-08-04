package start

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	tenantUpMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenant_up",
			Help: "Tenant availability status (1 = up, 0 = down)",
		},
		[]string{"domain"},
	)

	domainsMutex sync.Mutex
	domainsMap   = make(map[string]bool)
)

func init() {
	prometheus.MustRegister(tenantUpMetric)
}

func Main() {
	// Create a Kubernetes client to interact with the API server.
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	dynamicClient := dynamic.NewForConfigOrDie(config)

	// Set up a context for the dynamic client.
	ctx := context.Background()

	// Run a periodic function to fetch IngressRoute resources and update the tenant_up metric.
	go func() {
		for {
			// Fetch all namespaces from the Kubernetes API.
			namespaceList, err := dynamicClient.Resource(schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "namespaces",
			}).List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Println("Error fetching namespaces:", err)
			}

			// Create a channel to receive the domain information from Goroutines.
			domainChan := make(chan string)

			// Loop through all namespaces and launch a Goroutine for each namespace.
			for _, namespace := range namespaceList.Items {
				namespaceName := namespace.GetName()

				// Launch a Goroutine to fetch IngressRoute resources for the current namespace.
				go func(ns string) {
					gvr := schema.GroupVersionResource{
						Group:    "traefik.containo.us",
						Version:  "v1alpha1",
						Resource: "ingressroutes",
					}

					ingressRoutes, err := dynamicClient.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{})
					if err != nil {
						log.Printf("Error fetching IngressRoute resources in namespace %s: %v\n", ns, err)
					}

					numDiscovered := 0
					numWithAnnotations := 0

					// Extract domain values from the routes and send them to the channel.
					for _, ingressRoute := range ingressRoutes.Items {
						domain := getDomainFromAnnotations(ingressRoute)
						if err != nil {
							log.Printf("Error getting domain from annotations for IngressRoute in namespace %s: %v\n", ns, err)
							continue
						}

						if domain != "" {
							numWithAnnotations++
							domainChan <- domain
						}
						numDiscovered++
					}
					log.Printf("%d domains will be scraped from namespace %s out of a total of %d IngressRoutes.\n", numWithAnnotations, ns, numDiscovered)
				}(namespaceName)
			}

			// Close the channel after all Goroutines are done sending domain information.
			go func() {
				time.Sleep(30 * time.Second)
				close(domainChan)
			}()

			// Create a wait group to wait for all Goroutines to finish.
			var wg sync.WaitGroup

			// Launch Goroutines to make HTTP requests and set the metric concurrently.
			for domain := range domainChan {
				wg.Add(1)
				go func(d string) {
					defer wg.Done()
					status := getDomainStatus(d)
					tenantUpMetric.WithLabelValues(d).Set(status)

					// Lock the domainsMap for concurrent access.
					domainsMutex.Lock()

					// Store the domain in the domainsMap.
					domainsMap[d] = true

					// Unlock the domainsMap.
					domainsMutex.Unlock()
				}(domain)
			}

			// Wait for all Goroutines to finish before starting the next iteration.
			wg.Wait()

			// Lock the domainsMap for concurrent access.
			domainsMutex.Lock()

			// Check if any existing domains are not present in the domainChan.
			// If a domain is not present, remove its corresponding metric.
			for domain := range domainsMap {
				found := false
				for d := range domainChan {
					if domain == d {
						found = true
						break
					}
				}

				if !found {
					// Remove the metric for the domain.
					tenantUpMetric.Delete(prometheus.Labels{"domain": domain})

					// Remove the domain from the domainsMap.
					delete(domainsMap, domain)
				}
			}

			// Unlock the domainsMap.
			domainsMutex.Unlock()
		}
	}()

	// Create a new HTTP handler for Prometheus.
	http.Handle("/metrics", promhttp.Handler())

	// Start the HTTP server.
	log.Println("[INFO] Server listening on http://localhost:8080/metrics")
	http.ListenAndServe(":8080", nil)
}

func getDomainFromAnnotations(ingressRoute unstructured.Unstructured) string {
	annotations := ingressRoute.GetAnnotations()
	if annotations == nil {
		return ""
	}

	domain, domainFound := annotations["traefik-ingressroute-exporter/domain"]
	path, pathFound := annotations["traefik-ingressroute-exporter/path"]

	if !domainFound || !pathFound {
		// Ignore IngressRoute CRDs that don't have both annotations defined.
		return ""
	}

	domain = strings.TrimSuffix(domain, "/")  // Remove trailing slashes from the domain
	domain = strings.TrimSuffix(domain, "/*") // Remove wildcard paths from the domain
	path = strings.TrimPrefix(path, "/")      // Remove leading slashes from the path
	domain = domain + "/" + path              // Combine domain and path

	return domain
}

func getDomainStatus(domain string) float64 {
	// Make an HTTP GET request to the domain and check the response status code.
	resp, err := http.Get("https://" + domain)
	if err != nil {
		log.Printf("Error making HTTPS request to %s: %v\n", domain, err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// If the response status code is 200, return 1 (up).
		log.Printf("%s returned status code %d\n", domain, resp.StatusCode)
		return 1
	}

	// If the response status code is different than 200, return 0 (down).
	log.Printf("%s returned status code %d\n", domain, resp.StatusCode)
	return 0
}
