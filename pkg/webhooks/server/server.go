package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/resource"
	"github.com/apache/dubbo-kubernetes/pkg/config/validation"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/sail/pkg/config/kube/crd"
	"github.com/hashicorp/go-multierror"
	admissionv1 "k8s.io/api/admission/v1"
	kubeApiAdmissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
	"net/http"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// Expect AdmissionRequest to only include these top-level field names
	validFields = map[string]bool{
		"apiVersion": true,
		"kind":       true,
		"metadata":   true,
		"spec":       true,
		"status":     true,
	}
)

func init() {
	_ = admissionv1.AddToScheme(runtimeScheme)
	_ = kubeApiAdmissionv1beta1.AddToScheme(runtimeScheme)
}

type Options struct {
	Schemas      collection.Schemas
	DomainSuffix string
	Port         uint
	Mux          *http.ServeMux
}

// String produces a stringified version of the arguments for debugging.
func (o Options) String() string {
	buf := &bytes.Buffer{}

	_, _ = fmt.Fprintf(buf, "DomainSuffix: %s\n", o.DomainSuffix)
	_, _ = fmt.Fprintf(buf, "Port: %d\n", o.Port)

	return buf.String()
}

type Webhook struct {
	schemas      collection.Schemas
	domainSuffix string
}

// New creates a new instance of the admission webhook server.
func New(o Options) (*Webhook, error) {
	if o.Mux == nil {
		return nil, errors.New("expected mux to be passed, but was not passed")
	}
	wh := &Webhook{
		schemas:      o.Schemas,
		domainSuffix: o.DomainSuffix,
	}

	o.Mux.HandleFunc("/validate", wh.serveValidate)
	o.Mux.HandleFunc("/validate/", wh.serveValidate)

	return wh, nil
}

func (wh *Webhook) serveValidate(w http.ResponseWriter, r *http.Request) {
	serve(w, r, wh.validate)
}

func toAdmissionResponse(err error) *kube.AdmissionResponse {
	return &kube.AdmissionResponse{Result: &metav1.Status{Message: err.Error()}}
}

func (wh *Webhook) validate(request *kube.AdmissionRequest) *kube.AdmissionResponse {
	isDryRun := request.DryRun != nil && *request.DryRun
	addDryRunMessageIfNeeded := func(errStr string) error {
		err := fmt.Errorf("%s", errStr)
		if isDryRun {
			err = fmt.Errorf("%s (dry run)", err)
		}
		return err
	}
	switch request.Operation {
	case kube.Create, kube.Update:
	default:
		klog.Warningf("Unsupported webhook operation %v", addDryRunMessageIfNeeded(request.Operation))
		return &kube.AdmissionResponse{Allowed: true}
	}

	var obj crd.DubboKind
	if err := json.Unmarshal(request.Object.Raw, &obj); err != nil {
		klog.Infof("cannot decode configuration: %v", addDryRunMessageIfNeeded(err.Error()))
		return toAdmissionResponse(fmt.Errorf("cannot decode configuration: %v", err))
	}

	gvk := obj.GroupVersionKind()

	// "Version" is not relevant for Istio types; each version has the same schema. So do a lookup that does not consider
	// version. This ensures if a new version comes out and Istiod is not updated, we won't reject it.
	s, exists := wh.schemas.FindByGroupKind(resource.FromKubernetesGVK(&gvk))
	if !exists {
		klog.Infof("unrecognized type %v", addDryRunMessageIfNeeded(obj.GroupVersionKind().String()))
		return toAdmissionResponse(fmt.Errorf("unrecognized type %v", obj.GroupVersionKind()))
	}

	out, err := crd.ConvertObject(s, &obj, wh.domainSuffix)
	if err != nil {
		klog.Infof("error decoding configuration: %v", addDryRunMessageIfNeeded(err.Error()))
		return toAdmissionResponse(fmt.Errorf("error decoding configuration: %v", err))
	}

	warnings, err := s.ValidateConfig(*out)
	if err != nil {
		if _, f := out.Annotations[constants.AlwaysReject]; !f {
			// Hide error message if it was intentionally rejected (by our own internal call)
			klog.Infof("configuration is invalid: %v", addDryRunMessageIfNeeded(err.Error()))
		}
		return toAdmissionResponse(fmt.Errorf("configuration is invalid: %v", err))
	}

	if _, err := checkFields(request.Object.Raw, request.Kind.Kind, request.Namespace, obj.Name); err != nil {
		return toAdmissionResponse(err)
	}
	return &kube.AdmissionResponse{Allowed: true, Warnings: toKubeWarnings(warnings)}
}

func toKubeWarnings(warn validation.Warning) []string {
	if warn == nil {
		return nil
	}
	me, ok := warn.(*multierror.Error)
	if ok {
		res := []string{}
		for _, e := range me.Errors {
			res = append(res, e.Error())
		}
		return res
	}
	return []string{warn.Error()}
}

type admitFunc func(*kube.AdmissionRequest) *kube.AdmissionResponse

func serve(w http.ResponseWriter, r *http.Request, admit admitFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := kube.HTTPConfigReader(r); err == nil {
			body = data
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if len(body) == 0 {
		http.Error(w, "no body found", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "invalid Content-Type, want `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var reviewResponse *kube.AdmissionResponse
	var obj runtime.Object
	var ar *kube.AdmissionReview
	if out, _, err := deserializer.Decode(body, nil, obj); err != nil {
		reviewResponse = toAdmissionResponse(fmt.Errorf("could not decode body: %v", err))
	} else {
		ar, err = kube.AdmissionReviewKubeToAdapter(out)
		if err != nil {
			reviewResponse = toAdmissionResponse(fmt.Errorf("could not decode object: %v", err))
		} else {
			reviewResponse = admit(ar.Request)
		}
	}

	response := kube.AdmissionReview{}
	response.Response = reviewResponse
	var responseKube runtime.Object
	var apiVersion string
	if ar != nil {
		apiVersion = ar.APIVersion
		response.TypeMeta = ar.TypeMeta
		if response.Response != nil {
			if ar.Request != nil {
				response.Response.UID = ar.Request.UID
			}
		}
	}
	responseKube = kube.AdmissionReviewAdapterToKube(&response, apiVersion)
	resp, err := json.Marshal(responseKube)
	if err != nil {
		http.Error(w, fmt.Sprintf("could encode response: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(resp); err != nil {
		http.Error(w, fmt.Sprintf("could write response: %v", err), http.StatusInternalServerError)
	}
}

func checkFields(raw []byte, kind string, namespace string, name string) (string, error) {
	trial := make(map[string]json.RawMessage)
	if err := json.Unmarshal(raw, &trial); err != nil {
		klog.Errorf("cannot decode configuration fields: %v", err)
		return "yaml_decode_error", fmt.Errorf("cannot decode configuration fields: %v", err)
	}

	for key := range trial {
		if _, ok := validFields[key]; !ok {
			klog.Infof("unknown field %q on %s resource %s/%s",
				key, kind, namespace, name)
			return "invalid_resource", fmt.Errorf("unknown field %q on %s resource %s/%s",
				key, kind, namespace, name)
		}
	}

	return "", nil
}
