package metadata

const (
	PartialObjectMetadataListMimeParameters = "as=PartialObjectMetadataList;g=meta.k8s.io;v=v1"
	PartialObjectMetadataListMimeTypes      = "application/vnd.kubernetes.protobuf;" +
		PartialObjectMetadataListMimeParameters +
		",application/json;" + PartialObjectMetadataListMimeParameters
)
