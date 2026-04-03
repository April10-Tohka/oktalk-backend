"""
VAD gRPC 服务端
使用 silero-vad 进行语音活动检测，通过 gRPC 双向流与 Go 后端通信

生成 Python gRPC 代码命令（在 vad_service 目录执行）:
  python -m grpc_tools.protoc -I../proto --python_out=. --grpc_python_out=. ../proto/vad.proto
"""

import grpc
import concurrent.futures
import numpy as np
import torch
import vad_pb2
import vad_pb2_grpc
from concurrent import futures


# 加载 silero-vad 模型
model, utils = torch.hub.load(
    repo_or_dir='snakers4/silero-vad',
    model='silero_vad',
)
(get_speech_timestamps, save_audio, read_audio, VADIterator, collect_chunks) = utils


class VADServiceServicer(vad_pb2_grpc.VADServiceServicer):
    """实现 StreamingVAD 双向流 RPC"""

    def StreamingVAD(self, request_iterator, context):
        """
        双向流 RPC：
        - 持续接收 AudioChunk（PCM 字节流）
        - 内部做帧对齐（silero-vad 每次需要 512 采样点）
        - 检测 speech start/end，返回对应 VADEvent
        """
        # 创建 VADIterator，每次对话独立状态
        vad_iterator = VADIterator(
            model,
            threshold=0.5,
            sampling_rate=16000,
            min_silence_duration_ms=600,
            speech_pad_ms=100,
        )

        pcm_buffer = b""          # 字节缓冲，用于帧对齐
        speech_buffer = []        # 当前句子所有 float32 采样点列表
        frame_size = 512          # silero-vad 要求：16kHz 下 512 个采样点
        frame_bytes = frame_size * 2  # 16bit PCM，每采样 2 字节

        for chunk in request_iterator:
            if context.is_active() is False:
                break

            pcm_buffer += chunk.pcm_data

            # 按 512 采样点对齐处理
            while len(pcm_buffer) >= frame_bytes:
                frame_raw = pcm_buffer[:frame_bytes]
                pcm_buffer = pcm_buffer[frame_bytes:]

                # int16 → float32（silero-vad 需要 float32 [-1, 1]）
                audio_int16 = np.frombuffer(frame_raw, dtype=np.int16)
                audio_float32 = audio_int16.astype(np.float32) / 32768.0
                audio_tensor = torch.from_numpy(audio_float32)

                # 调用 VADIterator
                speech_dict = vad_iterator(audio_tensor, return_seconds=False)

                # 无论是否检测到语音，都把当前帧加入 speech_buffer
                # 这样 SPEECH_END 时可以拿到完整句子（含 pad）
                if speech_buffer is not None:
                    speech_buffer.append(audio_float32)

                if speech_dict is not None:
                    if 'start' in speech_dict:
                        # 检测到语音开始
                        # 重置 speech_buffer，只保留当前帧（pad 由 VADIterator 内部处理）
                        speech_buffer = [audio_float32]
                        yield vad_pb2.VADEvent(
                            type=vad_pb2.VADEvent.SPEECH_START,
                        )

                    elif 'end' in speech_dict:
                        # 检测到语音结束，构建完整句子 PCM
                        if speech_buffer:
                            full_audio = np.concatenate(speech_buffer, axis=0)
                            # float32 → int16 bytes
                            audio_int16_out = (full_audio * 32768.0).clip(-32768, 32767).astype(np.int16)
                            audio_bytes = audio_int16_out.tobytes()
                        else:
                            audio_bytes = b""

                        yield vad_pb2.VADEvent(
                            type=vad_pb2.VADEvent.SPEECH_END,
                            audio_data=audio_bytes,
                        )

                        # 重置状态
                        vad_iterator.reset_states()
                        speech_buffer = []


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    vad_pb2_grpc.add_VADServiceServicer_to_server(VADServiceServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("[VAD] gRPC server listening on port 50051")
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
