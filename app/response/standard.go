package response

type ResponseForm struct {
	Status   int32  `json:"status"`
	Services string `json:"service"`
	Message  string `json:"pesan"`
	Payload  any    `json:"response_payload"`
}

type ResponseMediaUpload struct {
	Status    int32  `json:"status"`
	Services  string `json:"service"`
	UrlUpload string `json:"upload_url"`
	Key       string `json:"key"`
	Type      string `json:"type"`
}

type UrlAndKey struct {
	UrlUpload string `json:"upload_url"`
	Key       string `json:"key"`
}

type ResponseMediaUploadBurst struct {
	Status    int32       `json:"status"`
	Services  string      `json:"service"`
	UrlAndKey []UrlAndKey `json:"url_and_key"`
	Type      string      `json:"type"`
}
