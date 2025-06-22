package metadata

import (
	"k8s.io/client-go/rest"
)

func ToPartialObjectMetadataList(req *rest.Request) {
	req.SetHeader("Accept", PartialObjectMetadataListMimeTypes)
}
