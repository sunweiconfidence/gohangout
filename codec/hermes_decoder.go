package codec

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/binary"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang/glog"
)

type HermesDecoder struct {
	MAGIC      []byte
	CRC_LENGTH int
}

func getCodecType(header []byte) string {
	offset := 0

	firstByte := int8(header[offset])
	if firstByte != -1 {
		length := int(binary.BigEndian.Uint32(header[offset:]))
		offset += 4
		offset += length
	} else {
		offset += 1
	}
	offset += 8 // skip bornTime
	offset += 4 // skip remaining retries

	firstByte = int8(header[offset])
	if firstByte != -1 {
		length := int(binary.BigEndian.Uint32(header[offset:]))
		offset += 4
		return string(header[offset : offset+length])
	}

	return ""
}

func (hd *HermesDecoder) Decode(s string) map[string]interface{} {
	value := []byte(s)
	offset := 0
	offset += 4

	//version := int(value[offset]) // version
	offset++

	//binary.BigEndian.Uint32(value[offset:]) // totalLength
	offset += 4

	headerLength := int(binary.BigEndian.Uint32(value[offset:]))
	offset += 4

	//bodyLength := binary.BigEndian.Uint32(value[offset:])
	offset += 4

	codecType := getCodecType(value[offset : offset+headerLength])

	offset += headerLength

	value = value[offset : len(value)-hd.CRC_LENGTH]

	codeAndCompress := strings.SplitN(codecType, ",", 2)
	if len(codeAndCompress) == 2 {
		if codeAndCompress[1] == "gzip" {
			reader, err := gzip.NewReader(bytes.NewReader(value))
			if err != nil {
				glog.Errorf("gzip decode hermes message error:%s", err)
				return nil
			}
			if value, err = ioutil.ReadAll(reader); err != nil && err != io.EOF {
				glog.Errorf("gzip decode hermes message error:%s", err)
				return nil
			}
		} else if strings.Contains(codeAndCompress[1], "deflater") {
			reader := flate.NewReader(bytes.NewReader(value))
			var err error
			if value, err = ioutil.ReadAll(reader); err != nil && err != io.EOF {
				glog.Errorf("gzip decode hermes message error:%s", err)
				return nil
			}
		} else {
			glog.Fatalf("%s unknown codec type", codecType)
		}
	}

	rst := make(map[string]interface{})
	rst["@timestamp"] = time.Now().UnixNano() / 1000000
	d := json.NewDecoder(bytes.NewReader(value))
	d.UseNumber()
	err := d.Decode(&rst)
	if err != nil {
		rst["message"] = string(value)
	}
	return rst
}
