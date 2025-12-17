package mb_serializer

type PublishPayloadProto struct {
	TableName string `json:"table_name"`
	Payload   []byte
}

type PublishPayloadJson struct {
	TableName string      `json:"table_name"`
	Payload   interface{} `json:"payload"`
}
