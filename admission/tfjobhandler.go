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
	//v1 "k8s.io/api/batch/v1"
	tfv1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

const (
	// TFReplicaTypePS is the type for parameter servers of distributed TensorFlow.
	TFReplicaTypePS commonv1.ReplicaType = "PS"

	// TFReplicaTypeWorker is the type for workers of distributed TensorFlow.
	// This is also used for non-distributed TensorFlow.
	TFReplicaTypeWorker commonv1.ReplicaType = "Worker"

	// TFReplicaTypeChief is the type for chief worker of distributed TensorFlow.
	// If there is "chief" replica type, it's the "chief worker".
	// Else, worker:0 is the chief worker.
	TFReplicaTypeChief commonv1.ReplicaType = "Chief"

	// TFReplicaTypeMaster is the type for master worker of distributed TensorFlow.
	// This is similar to chief, and kept just for backwards compatibility.
	TFReplicaTypeMaster commonv1.ReplicaType = "Master"

	// TFReplicaTypeEval is the type for evaluation replica in TensorFlow.
	TFReplicaTypeEval commonv1.ReplicaType = "Evaluator"
)
func IsWorker(typ commonv1.ReplicaType) bool {
	return typ == TFReplicaTypeWorker
}

// IsEvaluator returns true if the type is Evaluator.
func IsEvaluator(typ commonv1.ReplicaType) bool {
	return typ == TFReplicaTypeEval
}

func IsChief(typ commonv1.ReplicaType) bool {
	return typ == TFReplicaTypeChief
}

func IsMaster(typ commonv1.ReplicaType) bool {
	return typ == TFReplicaTypeMaster
}

func IsPS(typ commonv1.ReplicaType) bool {
	return typ == TFReplicaTypePS
}

func GetReplicaTypes(specs map[commonv1.ReplicaType]*commonv1.ReplicaSpec) []commonv1.ReplicaType {
	keys := make([]commonv1.ReplicaType, 0, len(specs))
	for k := range specs {
		keys = append(keys, k)
	}
	return keys
}

func TfjobAnnotationCheck(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
		reviewResponse := v1beta1.AdmissionResponse{}

		fmt.Println("check for enter experiment handler")

		tfjob := tfv1.TFJob{}		

		if err := json.Unmarshal(ar.Request.Object.Raw, &tfjob); err != nil {
			return ToAdmissionResponse(err) //msg: error		
		}
		tfjobns := tfjob.ObjectMeta.Namespace

		klog.Infof("TFJob is created in ns : %s", tfjobns)
	
		owners := tfjob.GetOwnerReferences()

		klog.Infof("TFJob owner : %s", owners)
		//jobKind := ""
		//jobName := ""
		// Search for Trial owner in object owner references
		// Trial is owned object if kind = Trial kind and API version = Trial API version

		var patch []patchOps

		am := map[string]string{
			"sidecar.istio.io/inject": "false",			
		}

		tfjobName := ""

		for _, owner := range owners {
			if owner.Kind == "Trial" && owner.APIVersion == "kubeflow.org/v1beta1" {
				tfjobName = tfjob.GetName()
			} 
		}
		
		klog.Infof("TFJob name : %s", tfjobName)

		if tfjobName != "" {
			replicaTypes := GetReplicaTypes(tfjob.Spec.TFReplicaSpecs)
				for _, replicaType := range replicaTypes{
					if IsChief(replicaType) {
						createPatch(&patch, "add", "/spec/tfReplicaSpecs/Chief/template/metadata/annotations", am)
					} else if IsMaster(replicaType){
						createPatch(&patch, "add", "/spec/tfReplicaSpecs/Master/template/metadata/annotations", am)
					} else if IsWorker(replicaType) {
						createPatch(&patch, "add", "/spec/tfReplicaSpecs/Worker/template/metadata/annotations", am)
					} else if IsEvaluator(replicaType) {
						createPatch(&patch, "add", "/spec/tfReplicaSpecs/Evaluator/template/metadata/annotations", am)
					} else if IsPS(replicaType) {
						createPatch(&patch, "add", "/spec/tfReplicaSpecs/PS/template/metadata/annotations", am)
					}	
				}	
		}


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