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
	//experimentsv1beta1 "github.com/kubeflow/katib/pkg/apis/controller/experiments/v1beta1"		
	"k8s.io/klog"	
	v1 "k8s.io/api/batch/v1"
)

func TrialSpecAnnotationCheck(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
		reviewResponse := v1beta1.AdmissionResponse{}

		fmt.Println("check for enter experiment handler")

		job := v1.Job{}		

		if err := json.Unmarshal(ar.Request.Object.Raw, &job); err != nil {
			return ToAdmissionResponse(err) //msg: error		
		}
	
		owners := job.GetOwnerReferences()

		//jobKind := ""
		//jobName := ""
		// Search for Trial owner in object owner references
		// Trial is owned object if kind = Trial kind and API version = Trial API version

		var patch []patchOps

		am := map[string]string{
			"sidecar.istio.io/inject": "false",			
		}

		for _, owner := range owners {
			if owner.Kind == "Trial" && owner.APIVersion == "kubeflow.org/v1beta1" {
				createPatch(&patch, "add", "/spec/Template/metadata/annotations", am)
			}
		}			

		// klog.Infof("job created in namespace : %s", jobNamespace)
				
		
	


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