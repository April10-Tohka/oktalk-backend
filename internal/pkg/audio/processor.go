// Package audio 提供音频处理工具
package audio

import (
	"bytes"
	"encoding/binary"
	"io"
)

// AudioProcessor 音频处理器
type AudioProcessor struct {
	validator *AudioValidator
	converter *Converter
}

// NewAudioProcessor 创建音频处理器
func NewAudioProcessor() *AudioProcessor {
	return &AudioProcessor{
		validator: NewAudioValidator(),
		converter: NewConverter(),
	}
}

// Process 处理音频
func (p *AudioProcessor) Process(reader io.Reader, format string) (*ProcessedAudio, error) {
	// 读取数据
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}
	data := buf.Bytes()

	// 验证
	result := p.validator.Validate(data, format)
	if !result.IsValid {
		return nil, &AudioError{Errors: result.Errors}
	}

	// 返回处理结果
	return &ProcessedAudio{
		Data:   data,
		Format: format,
		Size:   int64(len(data)),
	}, nil
}

// GetDuration 获取音频时长（毫秒）
func (p *AudioProcessor) GetDuration(data []byte, format string) (int, error) {
	// TODO: 实现获取音频时长
	// 根据不同格式解析音频头信息
	return 0, nil
}

// GetSampleRate 获取采样率
func (p *AudioProcessor) GetSampleRate(data []byte, format string) (int, error) {
	// TODO: 实现获取采样率
	return 0, nil
}

// ProcessedAudio 处理后的音频
type ProcessedAudio struct {
	Data       []byte `json:"data"`
	Format     string `json:"format"`
	Size       int64  `json:"size"`
	Duration   int    `json:"duration"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

// AudioError 音频错误
type AudioError struct {
	Errors []string
}

// Error 实现 error 接口
func (e *AudioError) Error() string {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return "audio processing error"
}

// WAVHeader WAV 文件头
type WAVHeader struct {
	ChunkID       [4]byte
	ChunkSize     uint32
	Format        [4]byte
	Subchunk1ID   [4]byte
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte
	Subchunk2Size uint32
}

// ParseWAVHeader 解析 WAV 文件头
func ParseWAVHeader(data []byte) (*WAVHeader, error) {
	if len(data) < 44 {
		return nil, &AudioError{Errors: []string{"invalid WAV file"}}
	}

	header := &WAVHeader{}
	reader := bytes.NewReader(data)
	
	if err := binary.Read(reader, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	return header, nil
}
