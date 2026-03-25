// Package wav 提供 RIFF/WAVE 容器解析，供讯飞 ISE（aue=raw, L16, 16k）等需要「去掉 WAV 头」的场景使用。
package wav

import (
	"encoding/binary"
	"fmt"
)

// StripToLinearPCM 从输入中取出「data」子块的原始 PCM 字节（小端线性 PCM，与容器内一致）。
//
// 若 buf 不是以 RIFF/WAVE 开头（例如客户端已上传裸 PCM、或 Apifox 传 body 无头），则原样返回，不报错。
// 若是 WAV 但损坏或找不到 data 块，则返回错误。
//
// 说明：讯飞英文评测常用 audio/L16;rate=16000；若 WAV 实为 44.1k/立体声，仅去头后采样率仍不对，
// 需在采集端重采样为 16k/mono/16bit，本包不负责重采样。
func StripToLinearPCM(buf []byte) ([]byte, error) {
	if len(buf) < 12 {
		return buf, nil
	}
	if string(buf[0:4]) != "RIFF" || string(buf[8:12]) != "WAVE" {
		return buf, nil
	}
	return extractDataChunk(buf)
}

func extractDataChunk(wave []byte) ([]byte, error) {
	// RIFF header 12 bytes, then chunks: id(4) + size(4) + data(size) [+ pad if size odd]
	offset := 12
	for offset+8 <= len(wave) {
		chunkID := string(wave[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(wave[offset+4 : offset+8]))
		offset += 8
		if chunkSize < 0 {
			return nil, fmt.Errorf("wav: invalid negative chunk size")
		}
		if chunkID == "data" {
			// 部分工具写入的 WAV 中 data 的 size 与文件实际长度不一致（或前面 chunk 解析错位导致
			// 把 PCM 误当成 size）。以「不超过剩余字节」为准截断到文件末尾。
			avail := len(wave) - offset
			if chunkSize > avail {
				chunkSize = avail
			}
			if chunkSize <= 0 {
				return nil, fmt.Errorf("wav: data chunk empty (offset=%d len=%d)", offset, len(wave))
			}
			return wave[offset : offset+chunkSize], nil
		}
		if chunkSize > len(wave)-offset {
			return nil, fmt.Errorf("wav: chunk %q size %d exceeds file (offset=%d len=%d)", chunkID, chunkSize, offset, len(wave))
		}
		offset += chunkSize
		if chunkSize&1 != 0 {
			offset++ // RIFF 2-byte alignment padding
		}
	}
	return nil, fmt.Errorf("wav: no data chunk")
}
