package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Serve the static HTML page
	r.StaticFile("/", "./index.html")

	// Serve the HLS stream
	r.GET("/hls/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		filepath := filepath.Join("hls_output", filename)
		c.File(filepath)
	})

	// Serve the HLS playlist
	r.GET("/playlist.m3u8", func(c *gin.Context) {
		playlist := `
        #EXTM3U
        #EXT-X-VERSION:3
        #EXT-X-TARGETDURATION:10
        #EXT-X-MEDIA-SEQUENCE:0
        #EXTINF:10,
        stream0.ts
        #EXTINF:10,
        stream1.ts
        #EXTINF:10,
        stream2.ts
        #EXT-X-ENDLIST
        `
		c.Data(http.StatusOK, "application/x-mpegURL", []byte(playlist))
	})

	// Endpoint to upload MP4 file and convert to .ts files
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.String(400, "Failed to get file: %v", err)
			return
		}

		// Save the uploaded file
		filename := filepath.Base(file.Filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.String(500, "Failed to save file: %v", err)
			return
		}

		// Create output directory for .ts files
		outputDir := "hls_output"
		os.MkdirAll(outputDir, os.ModePerm)

		// Run ffmpeg to convert MP4 to .ts files
		outputPattern := filepath.Join(outputDir, "stream%d.ts")
		cmd := exec.Command("ffmpeg", "-i", filename, "-preset", "ultrafast", "-c:v", "libx264", "-c:a", "aac", "-f", "segment", "-segment_time", "5", "-segment_format", "mpegts", outputPattern)

		// Capture ffmpeg output
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.String(500, "Failed to convert file: %v\nOutput: %s", err, string(output))
			return
		}

		// Generate playlist file
		playlistPath := filepath.Join(outputDir, "playlist.m3u8")
		playlist := "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n#EXT-X-MEDIA-SEQUENCE:0\n"
		files, err := filepath.Glob(filepath.Join(outputDir, "*.ts"))
		if err != nil {
			c.String(500, "Failed to read output files: %v", err)
			return
		}

		for _, file := range files {
			playlist += fmt.Sprintf("#EXTINF:10,\n%s\n", filepath.Base(file))
		}
		playlist += "#EXT-X-ENDLIST\n"

		if err := os.WriteFile(playlistPath, []byte(playlist), 0644); err != nil {
			c.String(500, "Failed to write playlist: %v", err)
			return
		}

		c.String(200, "File converted successfully")
	})

	// Start the server
	s := &http.Server{
		Addr:    ":8080",
		Handler: r,
		// ReadTimeout:    10 * time.Second,
		// WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}
