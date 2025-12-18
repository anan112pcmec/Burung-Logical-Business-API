package mb_cud_serializer

type PublishPayloadProto struct {
	TableName string `json:"table_name"`
	Protocol  string `json:"protocol"`
	Payload   []byte
}

type PublishPayloadJson struct {
	TableName string      `json:"table_name"`
	Protocol  string      `json:"protocol"`
	Payload   interface{} `json:"payload"`
}
