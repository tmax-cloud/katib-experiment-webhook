package admission

import (	
	"encoding/json"
	"fmt"	
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"		
	"k8s.io/klog"		
)

func PodAnnotationCheck(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

		//pod 정보 받아온다	
		reviewResponse := v1beta1.AdmissionResponse{}

		fmt.Println("check for enter experiment handler")

		pod := corev1.Pod{}		

		if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
			return ToAdmissionResponse(err) //msg: error		
		}
		//pod가 생성되는 namespace 
		podns := pod.ObjectMeta.Namespace

		klog.Infof("Pod is created in ns : %s", podns)
		
		//pod의 ownerreference 
		owners := pod.GetOwnerReferences()

		klog.Infof("Pod owner : %s", owners)		


		var patch []patchOps

		// 삽입할 annotation
		am := map[string]string{
			"sidecar.istio.io/inject": "false",			
		}

		podName := ""
		//ownerreference의 kind가 tfjob or pytorchjob인 pod 선택해서 이름 가져온다.
		for _, owner := range owners {
			if owner.Kind == "TFJob" && owner.APIVersion == "kubeflow.org/v1" {
				podName = pod.GetName()
			} else if owner.Kind == "PyTorchJob" && owner.APIVersion == "kubeflow.org/v1" {				
				podName = pod.GetName()	
			}
		}
		
		klog.Infof("POD name : %s", podName)

		//해당하는 pod에 annotation 삽입
		
		if podName != "" {
			createPatch(&patch, "add", "/metadata/annotations", am)
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