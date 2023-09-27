/*
Copyright 2025 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"fmt"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/handlers"
)

// TODO: Convert is a NOP converter that additionally stores the original APIVersion of each item in the annotation
// kcp.io/original-api-version. This is necessary for kcp with wildcard partial metadata list/watch requests.
// For example, if the request is for /clusters/*/apis/kcp.io/v1/widgets, and it's a partial metadata request, the
// server returns ALL widgets, regardless of their API version. But because this is a partial metadata request, the
// API version of the returned object is always meta.k8s.io/$version (could be v1 or v1beta1). Any client needing to
// modify or delete the returned object must know its exact API version. Therefore, we set this annotation with the
// actual original API version of the object. Clients can use it when constructing dynamic clients to guarantee they
// are using the correct API version.

func (c *crConverter) kcpConvertToVersion() (runtime.Object, error) {
	return nil, nil
}

func getObjectsToConvert(
	list *unstructured.UnstructuredList,
	desiredAPIVersion string,
	validVersions map[schema.GroupVersion]bool,
	requireValidVersion bool,
) ([]unstructured.Unstructured, error) {
	var objectsToConvert []unstructured.Unstructured
	for i := range list.Items {
		expectedGV := list.Items[i].GroupVersionKind().GroupVersion()
		if requireValidVersion && !validVersions[expectedGV] {
			return nil, fmt.Errorf("request to convert CR list failed, list index %d has invalid group/version: %s", i, expectedGV.String())
		}

		// First preserve the actual API version
		annotations := item.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[handlers.KCPOriginalAPIVersionAnnotation] = item.GetAPIVersion()
		item.SetAnnotations(annotations)

		// Now that we've preserved it, we can change it to the targetGV.
		item.SetGroupVersionKind(targetGV.WithKind(item.GroupVersionKind().Kind))

		// Only sent item for conversion, if the apiVersion is different
		if list.Items[i].GetAPIVersion() != desiredAPIVersion {
			objectsToConvert = append(objectsToConvert, list.Items[i])
		}
	}
	return objectsToConvert, nil
}

// isEmptyUnstructuredObject returns true if in is an empty unstructured object, i.e. an unstructured object that does
// not have any field except apiVersion and kind.
func isEmptyUnstructuredObject(in runtime.Object) bool {
	u, ok := in.(*unstructured.Unstructured)
	if !ok {
		return false
	}
	if len(u.Object) != 2 {
		return false
	}
	if _, ok := u.Object["kind"]; !ok {
		return false
	}
	if _, ok := u.Object["apiVersion"]; !ok {
		return false
	}
	return true
}

// validateConvertedObject checks that ObjectMeta fields match, with the exception of
// labels and annotations.
func validateConvertedObject(in, out *unstructured.Unstructured) error {
	if e, a := in.GetKind(), out.GetKind(); e != a {
		return fmt.Errorf("must have the same kind: %v != %v", e, a)
	}
	if e, a := in.GetName(), out.GetName(); e != a {
		return fmt.Errorf("must have the same name: %v != %v", e, a)
	}
	if e, a := in.GetNamespace(), out.GetNamespace(); e != a {
		return fmt.Errorf("must have the same namespace: %v != %v", e, a)
	}
	if e, a := in.GetUID(), out.GetUID(); e != a {
		return fmt.Errorf("must have the same UID: %v != %v", e, a)
	}
	return nil
}

// restoreObjectMeta copies metadata from original into converted, while preserving labels and annotations from converted.
func restoreObjectMeta(original, converted *unstructured.Unstructured) error {
	cm, found := converted.Object["metadata"]
	om, previouslyFound := original.Object["metadata"]
	switch {
	case !found && !previouslyFound:
		return nil
	case previouslyFound && !found:
		return fmt.Errorf("missing metadata in converted object")
	case !previouslyFound && found:
		om = map[string]interface{}{}
	}

	convertedMeta, ok := cm.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid metadata of type %T in converted object", cm)
	}
	originalMeta, ok := om.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid metadata of type %T in input object", om)
	}

	result := converted
	if previouslyFound {
		result.Object["metadata"] = originalMeta
	} else {
		result.Object["metadata"] = map[string]interface{}{}
	}
	resultMeta := result.Object["metadata"].(map[string]interface{})

	for _, fld := range []string{"labels", "annotations"} {
		obj, found := convertedMeta[fld]
		if !found || obj == nil {
			delete(resultMeta, fld)
			continue
		}

		convertedField, ok := obj.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid metadata.%s of type %T in converted object", fld, obj)
		}
		originalField, ok := originalMeta[fld].(map[string]interface{})
		if !ok && originalField[fld] != nil {
			return fmt.Errorf("invalid metadata.%s of type %T in original object", fld, originalMeta[fld])
		}

		somethingChanged := len(originalField) != len(convertedField)
		for k, v := range convertedField {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("metadata.%s[%s] must be a string, but is %T in converted object", fld, k, v)
			}
			if originalField[k] != interface{}(v) {
				somethingChanged = true
			}
		}

		if somethingChanged {
			stringMap := make(map[string]string, len(convertedField))
			for k, v := range convertedField {
				stringMap[k] = v.(string)
			}
			var errs field.ErrorList
			if fld == "labels" {
				errs = metav1validation.ValidateLabels(stringMap, field.NewPath("metadata", "labels"))
			} else {
				errs = apivalidation.ValidateAnnotations(stringMap, field.NewPath("metadata", "annotation"))
			}
			if len(errs) > 0 {
				return errs.ToAggregate()
			}
		}

		resultMeta[fld] = convertedField
	}

	return nil
}
