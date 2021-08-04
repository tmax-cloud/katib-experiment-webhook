package admission

import (
	//"context"
	"encoding/json"
	"fmt"
	//"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	//"strconv"
	//"runtime/debug"	
	//"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/api/admission/v1beta1"
	//corev1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//rbacv1 "k8s.io/api/rbac/v1"
	experimentsv1beta1 "github.com/kubeflow/katib/pkg/apis/controller/experiments/v1beta1"		
	"k8s.io/klog"	
)

func TrialSpecAnnotationCheck(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
		reviewResponse := v1beta1.AdmissionResponse{}

		fmt.Println("check for enter experiment handler")

		ms := experimentsv1beta1.Experiment{}		
    
		if err := json.Unmarshal(ar.Request.Object.Raw, &ms); err != nil {
			return ToAdmissionResponse(err) //msg: error
		}
		nsofexperiment := ms.ObjectMeta.Namespace

		klog.Infof("experiment created in namespace : %s", nsofexperiment)
		
		
		kind := ms.Spec.TrialTemplate.TrialSpec.GetKind()

		annotationcheck := ms.Spec.TrialTemplate.TrialSpec.GetAnnotations()

		annotationInject := "sidecar.istio.io/inject: false"

		am := map[string]string{
			"sidecar.istio.io/inject": "false",			
		}

		klog.Infof("kind is : %s", kind)

		var patch []patchOps												

		if kind == "Job"{
			if annotationcheck == nil{
				ms.Spec.TrialTemplate.TrialSpec.SetAnnotations(am)
			} else{
				createPatch(&patch, "add", "/spec/trialTemplate/trialSpec/spec/template/metadata/annotations", annotationInject)
			}
		} else if kind == "PyTorchJob"{
			if annotationcheck == nil{
				ms.Spec.TrialTemplate.TrialSpec.SetAnnotations(am)
			} else{
				createPatch(&patch, "add", "/spec/trialTemplate/trialSpec/pytorchReplicaSpecs/Worker/template/metadata/annotations", annotationInject)
				createPatch(&patch, "add", "/spec/trialTemplate/trialSpec/pytorchReplicaSpecs/Master/template/metadata/annotations", annotationInject)
			}			  					
		} else if kind == "TFJob"{
			if annotationcheck == nil{
				ms.Spec.TrialTemplate.TrialSpec.SetAnnotations(am)
			} else{
				createPatch(&patch, "add", "/spec/trialTemplate/trialSpec/tfReplicaSpecs/PS/template/metadata/annotations", annotationInject)
				createPatch(&patch, "add", "/spec/trialTemplate/trialSpec/tfReplicaSpecs/Worker/template/metadata/annotations", annotationInject)
			}							
		}		

	//klog.Infof("check data for ms.Spec : %s", ms.Spec)

		if patchData, err := json.Marshal(patch); err != nil {
			return ToAdmissionResponse(err) //msg: error
		} else {
			klog.Infof("JsonPatch=%s", string(patchData))
			reviewResponse.Patch = patchData
		}

	// v1beta1 pkg에 저장된 patchType (const string)을 Resp에 저장
		pt := v1beta1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt
		reviewResponse.Allowed = true

		return &reviewResponse

}

