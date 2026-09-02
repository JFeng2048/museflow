#!/bin/bash
# Ollama Embedding 模型初始化脚本

echo "=========================================="
echo "Ollama Embedding 模型初始化"
echo "=========================================="

# 等待服务启动
echo "等待 Ollama 服务启动..."
sleep 5

# 检查服务状态
if ! curl -s http://localhost:11434 > /dev/null; then
    echo "错误: Ollama 服务未启动，请先运行 docker compose up -d"
    exit 1
fi

echo ""
echo "服务已就绪，开始下载模型..."
echo ""

# 下载 Qwen3-Embedding 4B 模型（支持 1536 维，中英文能力强）
echo ">>> 下载 Qwen3-Embedding 4B 模型 (支持 1536 维) <<<"
docker exec -it ollama-embed ollama pull qwen3-embedding:4b

echo ""
echo ">>> 下载备用模型 nomic-embed-text (768 维) <<<"
docker exec -it ollama-embed ollama pull nomic-embed-text

echo ""
echo "=========================================="
echo "已安装模型列表："
echo "=========================================="
docker exec -it ollama-embed ollama list

echo ""
echo "=========================================="
echo "初始化完成！"
echo ""
echo "使用示例："
echo "  # 生成 1536 维向量"
echo "  curl -X POST http://localhost:11434/api/embeddings \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -d '{\"model\": \"qwen3-embedding:4b\", \"prompt\": \"你的文本\"}'"
echo ""
echo "  # 注意：qwen3-embedding:4b 最大支持 2560 维"
echo "  # 如需 1536 维，请在请求中添加 \\\"dimensions\\\": 1536"
echo "=========================================="
