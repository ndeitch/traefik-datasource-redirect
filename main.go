package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)


type Config struct {
	Header 		 string `json:"header,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	DataSourceURL string `json:"dataSourceURL,omitempty"`
	Enabled bool `json:"enabled,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func CreateConfig() *Config {
	return &Config{}
}

// Plugin a simple provider plugin.
type Plugin struct {
	header 		 string
	serviceName  string
	dataSourceURL   string
	namespace 	 string
	next 			 http.Handler
	k8sClient 	 *kubernetes.Clientset
}

// New creates a new Provider plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if !config.Enabled {
		return next, nil
	}

	if config.Header == "" {
		return nil, fmt.Errorf("header must not be empty")
	}

	if config.ServiceName == "" {
		return nil, fmt.Errorf("serviceName must not be empty")
	}

	if config.DataSourceURL == "" {
		return nil, fmt.Errorf("dataSourceURL must not be empty")
	}

	clientConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	
	k8sClient, err := kubernetes.NewForConfig(clientConfig)

	if err != nil {
		return nil, err
	}

	return &Plugin{
		header: config.Header,
		serviceName: config.ServiceName,
		dataSourceURL: config.DataSourceURL,
		next: next,
		k8sClient: k8sClient,
		namespace: config.Namespace,
	}, nil
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if header := r.Header.Get(p.header); header == "" {
		p.RespondWithJSON(w)
		return
	}
}

func (p *Plugin) GetDataFromServicesByName() {
	pods, err := p.k8sClient.CoreV1().Pods(p.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.serviceName),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	for _, pod := range pods.Items {
		fmt.Printf("Pod name %s\n", pod.Name)
	}
}


func (p *Plugin) RespondWithJSON(w http.ResponseWriter) {
	response, _ := json.Marshal(ErrorResponse{Message: fmt.Sprintf("Missing required header: %s", p.header)})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write(response)
}