package crawlpb

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// 本文件在包初始化时构造 proto/crawl/crawl.proto 的 FileDescriptorProto 并序列化为
// rawDesc，等价于 protoc 生成的二进制描述符，使消息可正常序列化/反序列化。
// 当仓库可获取 protoc 时，可用 scripts/gen-proto.sh 重新生成标准的 crawl.pb.go 替换之。

func mustMarshalCrawlFile() []byte {
	fd := buildCrawlFileDescriptor()
	b, err := proto.Marshal(fd)
	if err != nil {
		panic("crawl: 序列化 FileDescriptorProto 失败: " + err.Error())
	}
	// DescBuilder.RawDescriptor 期望未压缩的 FileDescriptorProto 序列化字节。
	return b
}

func buildCrawlFileDescriptor() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("proto/crawl/crawl.proto"),
		Package: proto.String("crawl"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			msg("HealthRequest",
				fld("healthy", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
			),
			msg("HealthResponse",
				fld("healthy", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("service", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("version", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("uptime_seconds", 4, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, ""),
				fld("auth_enabled", 5, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
			),
			msg("CrawlerOptions",
				fld("timeout", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, ""),
				fld("user_agent", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("bypass_cache", 3, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("remove_overlay_elements", 4, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("simulate_user", 5, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("magic", 6, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("locale", 7, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
			),
			msg("LLMConfig",
				fld("api_key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("base_url", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("model", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("temperature", 4, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, ""),
				fld("max_tokens", 5, descriptorpb.FieldDescriptorProto_TYPE_INT32, ""),
				fld("request_timeout", 6, descriptorpb.FieldDescriptorProto_TYPE_INT32, ""),
			),
			msg("ExtractSchemaField",
				fld("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("description", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("type", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("required", 4, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("items_json", 5, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
			),
			msg("CrawlRequest",
				fld("url", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("options", 2, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".crawl.CrawlerOptions"),
				fld("wait_for", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
			),
			msg("CrawlResponse",
				fld("success", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("url", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("markdown", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("status_code", 4, descriptorpb.FieldDescriptorProto_TYPE_INT32, ""),
				fld("error_code", 5, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("error_message", 6, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("error_retryable", 7, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("elapsed_ms", 8, descriptorpb.FieldDescriptorProto_TYPE_INT32, ""),
			),
			msg("ExtractRequest",
				fld("url", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("instruction", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fldRepeated("schema_fields", 3, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".crawl.ExtractSchemaField"),
				fld("llm", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".crawl.LLMConfig"),
				fld("options", 5, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".crawl.CrawlerOptions"),
				fld("extraction_timeout", 6, descriptorpb.FieldDescriptorProto_TYPE_INT32, ""),
			),
			msg("ExtractResponse",
				fld("success", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("url", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("markdown", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("data_json", 4, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("error_code", 5, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("error_message", 6, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				fld("error_retryable", 7, descriptorpb.FieldDescriptorProto_TYPE_BOOL, ""),
				fld("elapsed_ms", 8, descriptorpb.FieldDescriptorProto_TYPE_INT32, ""),
				fld("model", 9, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
			),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("CrawlService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					method("Health", ".crawl.HealthRequest", ".crawl.HealthResponse", false),
					method("Crawl", ".crawl.CrawlRequest", ".crawl.CrawlResponse", false),
					method("Extract", ".crawl.ExtractRequest", ".crawl.ExtractResponse", false),
				},
			},
		},
	}
}

func msg(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: proto.String(name), Field: fields}
}

func fld(name string, number int32, typ descriptorpb.FieldDescriptorProto_Type, typeName string) *descriptorpb.FieldDescriptorProto {
	f := &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(number),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     typ.Enum(),
		JsonName: proto.String(jsonName(name)),
	}
	if typeName != "" {
		f.TypeName = proto.String(typeName)
	}
	return f
}

func fldRepeated(name string, number int32, typ descriptorpb.FieldDescriptorProto_Type, typeName string) *descriptorpb.FieldDescriptorProto {
	f := fld(name, number, typ, typeName)
	f.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	return f
}

func method(name, in, out string, serverStream bool) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:       proto.String(name),
		InputType:  proto.String(in),
		OutputType: proto.String(out),
		ClientStreaming: proto.Bool(false),
		ServerStreaming: proto.Bool(serverStream),
	}
}

// jsonName 依据 proto 的 JSON 命名规则：snake_case -> camelCase。
func jsonName(s string) string {
	var b []byte
	up := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			up = true
			continue
		}
		if up {
			if c >= 'a' && c <= 'z' {
				c = c - 'a' + 'A'
			}
			up = false
		}
		b = append(b, c)
	}
	return string(b)
}
