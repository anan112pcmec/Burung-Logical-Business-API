package mb_cud_serializer

import "github.com/anan112pcmec/Burung-backend-1/app/helper"

func NewProtoPayload() *PublishPayloadProto {
	return &PublishPayloadProto{
		Protocol: "protobuf",
	}
}

func (p *PublishPayloadProto) SetTableName(table string) *PublishPayloadProto {
	p.TableName = table
	return p
}

func (p *PublishPayloadProto) SetProtocol(protocol string) *PublishPayloadProto {
	p.Protocol = protocol
	return p
}

func (p *PublishPayloadProto) SetPayload(payload []byte) *PublishPayloadProto {
	p.Payload = payload
	return p
}

// =====================
// JSON PAYLOAD
// =====================

func NewJsonPayload() *PublishPayloadJson {
	return &PublishPayloadJson{
		Protocol: "json",
	}
}

func (j *PublishPayloadJson) SetTableName(table string) *PublishPayloadJson {
	j.TableName = table
	return j
}

func (j *PublishPayloadJson) SetProtocol(protocol string) *PublishPayloadJson {
	j.Protocol = protocol
	return j
}

func (j *PublishPayloadJson) SetPayload(payload interface{}) *PublishPayloadJson {
	jsonParse := helper.StructToJSONMap(payload)
	j.Payload = jsonParse
	return j
}
