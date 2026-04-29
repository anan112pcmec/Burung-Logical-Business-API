package mb_cud_serializer

type PublishPayloadProto struct {
	TableName string `json:"table_name"`
	Protocol  string `json:"protocol"`
	Role      string `json:"role"`
	Payload   []byte
}

type PublishPayloadJson struct {
	TableName string      `json:"table_name"`
	Protocol  string      `json:"protocol"`
	Role      string      `json:"role"`
	Payload   interface{} `json:"payload"`
}
