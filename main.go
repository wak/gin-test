package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func handleUpload(c *gin.Context) {
	start := time.Now()
	reader, err := c.Request.MultipartReader()
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Request (%s)", err.Error())
		fmt.Println(err.Error())
		return
	}

	var part *multipart.Part
	for {
		part, err = reader.NextPart()
		if err != nil {
			c.String(http.StatusBadRequest, "Bad Request (%s)", err.Error())
			return
		}
		if part.FormName() == "file" {
			break
		}
	}
	// fmt.Printf("FileName = %s\n", part.FileName())
	var totalBytes uint64
	var sha256sum string
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		hasher := sha256.New()
		buf := make([]byte, 10*1024*1024)
		for {
			nread, err := part.Read(buf)
			if nread > 0 {
				totalBytes += uint64(nread)
				hasher.Write(buf[:nread])
				pipeWriter.Write(buf[:nread])
			}
			if err != nil {
				if err == io.EOF {
					sha256sum = hex.EncodeToString(hasher.Sum(nil))
					pipeWriter.Close()
				} else {
					pipeWriter.CloseWithError(io.ErrUnexpectedEOF)
				}
				break
			}
		}
	}()

	buf2 := make([]byte, 10*1024*1024)
	for {
		_, err := pipeReader.Read(buf2)
		if err != nil {
			if err == io.EOF {
				elapsed := float64(time.Now().Sub(start)) / float64(time.Second)
				mBps := (float64(totalBytes) * 8 / 1024 / 1024) / float64(elapsed)
				c.JSON(http.StatusOK, gin.H{"size": totalBytes, "sha256": sha256sum, "mbps": mBps})
				return
			} else {
				c.String(http.StatusBadRequest, "Upload failed.")
			}
		}
	}
}

func main() {
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB しかし、使わない。
	router.Use(func(c *gin.Context) {
		c.Set("startedAt", time.Now())
		c.Next()
	})
	router.Static("/", "./public")
	router.POST("/upload", handleUpload)
	router.Run(":8080")
}