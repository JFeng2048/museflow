// Command protogen 在无 protoc 的环境下生成 gRPC / protobuf Go 代码。
//
// 原理：用纯 Go 解析器（protoparse）把 .proto 编译为 FileDescriptorSet，
// 再以 CodeGeneratorRequest 的形式调用已安装的 protoc-gen-go /
// protoc-gen-go-grpc 插件（位于 GOPATH/bin），与 protoc 的插件协议完全兼容。
//
// 独立成模块（不属于 go.work）是为了不把开发工具依赖带入
// proto / 各服务的 go.mod。
//
// 用法（在仓库根目录执行）：
//
//	cd tools/protogen && go run . -I . proto/user/user.proto proto/crawl/crawl.proto
//
// 参数：
//
//	-I    import 根目录，默认 "."
//	其余参数为待生成的 .proto 路径（相对 -I）
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	importRoot := flag.String("I", ".", "proto import 根目录")
	flag.Parse()

	targets := flag.Args()
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "错误：未指定 .proto 文件")
		os.Exit(1)
	}

	if err := generate(*importRoot, targets); err != nil {
		fmt.Fprintf(os.Stderr, "生成失败: %v\n", err)
		os.Exit(1)
	}
}

func generate(importRoot string, targets []string) error {
	// 1. 解析 .proto（含依赖）
	p := protoparse.Parser{
		ImportPaths:           []string{importRoot},
		IncludeSourceCodeInfo: true,
	}
	fds, err := p.ParseFiles(targets...)
	if err != nil {
		return fmt.Errorf("解析 proto 失败: %w", err)
	}

	// 2. 收集全部 FileDescriptorProto（依赖在前）
	var all []*descriptorpb.FileDescriptorProto
	seen := map[string]bool{}
	for _, fd := range fds {
		collect(fd, &all, seen)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: targets,
		// 与 scripts/gen-proto.sh 保持一致的生成选项
		Parameter: proto.String("paths=source_relative"),
		ProtoFile: all,
		CompilerVersion: &pluginpb.Version{
			Major: proto.Int32(5), Minor: proto.Int32(29), Patch: proto.Int32(3),
		},
	}

	// 3. 分别生成消息与服务代码
	for _, plugin := range []string{"protoc-gen-go", "protoc-gen-go-grpc"} {
		if err := invoke(plugin, req, importRoot); err != nil {
			return err
		}
	}
	return nil
}

// collect 先递归收集依赖，再收集自身，保证拓扑顺序。
func collect(fd *desc.FileDescriptor, out *[]*descriptorpb.FileDescriptorProto, seen map[string]bool) {
	if seen[fd.GetName()] {
		return
	}
	seen[fd.GetName()] = true
	for _, dep := range fd.GetDependencies() {
		collect(dep, out, seen)
	}
	*out = append(*out, fd.AsFileDescriptorProto())
}

// invoke 序列化请求并调用插件，将响应内容写入文件。
func invoke(plugin string, req *pluginpb.CodeGeneratorRequest, importRoot string) error {
	in, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	cmd := exec.Command(plugin)
	cmd.Stdin = bytes.NewReader(in)
	cmd.Stderr = os.Stderr
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行插件 %s 失败: %w（请先 go install google.golang.org/protobuf/cmd/protoc-gen-go@latest）", plugin, err)
	}

	resp := &pluginpb.CodeGeneratorResponse{}
	if err := proto.Unmarshal(out.Bytes(), resp); err != nil {
		return fmt.Errorf("解析插件 %s 响应失败: %w", plugin, err)
	}
	if resp.GetError() != "" {
		return fmt.Errorf("插件 %s 返回错误: %s", plugin, resp.GetError())
	}

	for _, f := range resp.File {
		dest := filepath.Join(importRoot, filepath.FromSlash(f.GetName()))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, []byte(f.GetContent()), 0o644); err != nil {
			return fmt.Errorf("写入 %s 失败: %w", dest, err)
		}
		fmt.Printf("已生成: %s\n", strings.TrimPrefix(dest, importRoot+string(filepath.Separator)))
	}
	return nil
}
