package conversion

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/handlers"
)

// setOriginalAPIVersion stores the original APIVersion of an itme in
// the annotation kcp.io/original-api-version.
//
// This is necessary for kcp with wildcard partial metadata list/watch
// requests. For example, if the request is for
// /clusters/*/apis/kcp.io/v1/widgets, and it's a partial metadata
// request, the server returns ALL widgets, regardless of their API
// version. But because this is a partial metadata request, the API
// version of the returned object is always meta.k8s.io/$version (could
// be v1 or v1beta1). Any client needing to modify or delete the
// returned object must know its exact API version. Therefore, we set
// this annotation with the actual original API version of the object.
// Clients can use it when constructing dynamic clients to guarantee
// they are using the correct API version.
func setOriginalAPIVersion(obj *unstructured.Unstructured, originalAPIVersion string, targetGV schema.GroupVersion) {
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}
	ann[handlers.KCPOriginalAPIVersionAnnotation] = originalAPIVersion
	obj.SetAnnotations(ann)
	obj.SetGroupVersionKind(targetGV.WithKind(obj.GetKind()))
}

// Factory is the interface for CRConverterFactory.
//
// KCP passes its own noop CRConverterFactory to the
// apiextension-apiserver.
type Factory interface {
	// NewConverter returns a CRConverter capable of converting crd's versions.
	//
	// For proper conversion, the returned CRConverter must be used via NewDelegatingConverter.
	//
	// When implementing a CRConverter, you do not need to: test for valid API versions or no-op
	// conversions, handle field selector logic, or handle scale conversions; these are all handled
	// via NewDelegatingConverter.
	NewConverter(crd *apiextensionsv1.CustomResourceDefinition) (runtime.ObjectConvertor, runtime.ObjectConvertor, error)
}

// Ensure that the upstream implementation satisfies the interface. If
// it doesn't the interface and the CRConverterFactory in the KCP code
// has to be updated.
var _ Factory = &CRConverterFactory{}

// Type alias to export the interface
// TODO(ntnn): Only needed for the schemaBasedConverter in KCP.
type CRConverterInterface crConverterInterface

// Export safeConverterWrapper for KCP to use.
func NewSafeConverterWrapper(unsafe runtime.ObjectConvertor) runtime.ObjectConvertor {
	return &safeConverterWrapper{
		unsafe: unsafe,
	}
}

func NewCRConverter(converter CRConverterInterface) runtime.ObjectConvertor {
	return &crConverter{
		converter: converter,
	}
}

func NewNOPConverter() runtime.ObjectConvertor {
	return NewCRConverter(&nopConverter{})
}

// TODO(ntnn): A very similar function exists now in webhook_conversion.go.
func kcpGetObjectsToConvert(
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

		// Only sent item for conversion, if the apiVersion is different
		if list.Items[i].GetAPIVersion() != desiredAPIVersion {
			objectsToConvert = append(objectsToConvert, list.Items[i])
		}
	}
	return objectsToConvert, nil
}

// restoreObjectMeta copies metadata from original into converted, while preserving labels and annotations from converted.
func kcpRestoreObjectMeta(original, converted *unstructured.Unstructured) error {
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
