package schema

type (
	DataPopulationProps struct {
		LocalPath string
		ObjectKey string
	}
)

type (
	BatchInsertDocumentResponse struct {
		ObjectKey    string `json:"object_key"`
		OriginalName string `json:"original_name"`
		Error        string `json:"error,omitempty"`
	}
)
