package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"	
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	expev1 "github.com/kubeflow/katib/pkg/apis/v1beta1"

	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
	defaulter     = runtime.ObjectDefaulter(runtimeScheme)
)

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

const (
	admissionWebhookAnnotationInjectKey = "sidecar.istio.io/inject"
)

// Webhook Server parameters
type WhSvrParameters struct {
	port                    int    // webhook server port
	certFile                string // path to the x509 certificate for https
	keyFile                 string // path to the x509 private key matching `CertFile`	
}

type WebhookServer struct {	
	server              *http.Server
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	corev1.AddToScheme(runtimeScheme)
	admissionregistrationv1.AddToScheme(runtimeScheme)
	corev1.AddToScheme(runtimeScheme)
}

func (ws WebhookServer) Serve(responseWriter http.ResponseWriter, request *http.Request) {
	var requestBody []byte
	if request.Body != nil {
		if data, err := ioutil.ReadAll(request.Body); err == nil {
			requestBody = data
		}
	}

	if len(requestBody) == 0 {
		log.Error("Empty Body")
		http.Error(responseWriter, "Empty Body", http.StatusBadRequest)
	}

	var admissionResponse *admissionv1.AdmissionResponse
	originAdmissionReview := admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(requestBody, nil, &originAdmissionReview); err != nil {
		log.Errorf("Can't decode request body: %v", err)
		admissionResponse = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = ws.mutate(&originAdmissionReview)
	}

	typeMeta := metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1",
	}

	mutatedAdmissionReview := admissionv1.AdmissionReview{TypeMeta: typeMeta}
	mutatedAdmissionReview.Response = admissionResponse
	if originAdmissionReview.Request != nil {
		mutatedAdmissionReview.Response.UID = originAdmissionReview.Request.UID
	}

	data, err := json.Marshal(mutatedAdmissionReview)
	if err != nil {
		log.Errorf("Can't encode response : %v", err)
		http.Error(responseWriter, fmt.Sprintf("Can't encode response : %v", err), http.StatusInternalServerError)
	}

	log.Infof("Ready to write response")
	if _, err := responseWriter.Write(data); err != nil {
		log.Errorf("Can't Write Response : %v", err)
		http.Error(responseWriter, fmt.Sprintf("Can't write response : %v", err), http.StatusInternalServerError)
	}
}

func (ws WebhookServer) mutate(admissionReview *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	request := admissionReview.Request
	var experiment expev1.Experiment
	if err := json.Unmarshal(request.Object.Raw, &experiment); err != nil {
		log.Errorf("Couldn't unmarshall raw object : %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionReview for Kind=%v | Namespace=%v | Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		request.Kind, request.Namespace, request.Name, experiment.Name, request.UID, request.Operation, request.UserInfo)

	if !isMutationTarget(ignoredNamespaces, &experiment.ObjectMeta) {
		log.Infof("Skip mutation for %s/%s", experiment.Namespace, experiment.Namespace)
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	
	annotations := map[string]string{admissionWebhookAnnotationInjectKey: "false"}
	patchBytes, err := createPatch(&experiment, annotations)

	if err != nil {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionResponse JSONPatch = %v\n", string(patchBytes))
	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func createPatch(experiment *expev1.Experiment, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation	
	patch = append(patch, updateAnnotation(experiment.spec.trialTemplate.trialSpec.spec.template.metadata.annotations, annotations)...)

	return json.Marshal(patch)
}

func updateAnnotation(experimentAnnotation map[string]string, annotations map[string]string) (patch []patchOperation) {
	for k, v := range annotations {
		if experimentAnnotation = nil  {
			patch = append(patch, patchOperation{
				Op:   "replace",
				Path: "/spec/trialTemplate/trialSpec/spec/template/metadata/annotations",
				Value: map[string]string{
					k: v,
				},
			})
		}
	}
	return patch
}

func isMutationTarget(ignoreNamespaces []string, metadata *metav1.ObjectMeta) bool {
	for _, namespace := range ignoredNamespaces {
		if metadata.Namespace == namespace {
			log.Infof("Skip mutation for %s namespace")
			return false
		}
	}
	
}
