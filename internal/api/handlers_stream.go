package api

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")

	file, err := s.tg.Storage().GetFile(fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))

	if err := s.tg.DownloadFile(r.Context(), fileID, w); err != nil {
		s.logger.Error("download failed", zap.String("file_id", fileID), zap.Error(err))
		return
	}
}

func (s *Server) handleStreamFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")

	file, err := s.tg.Storage().GetFile(fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	// ETag based on file ID and size for cache validation
	etag := fmt.Sprintf(`"%x"`, md5.Sum([]byte(fmt.Sprintf("%s-%d", fileID, file.Size))))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", etag)

	// Handle conditional request
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	rangeHeader := r.Header.Get("Range")

	// If no Range header, return 200 with Content-Length so the player
	// knows the file size and can make targeted Range requests.
	if rangeHeader == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
		w.WriteHeader(http.StatusOK)
		if err := s.tg.DownloadFile(r.Context(), fileID, w); err != nil {
			s.logger.Error("download failed",
				zap.String("file_id", fileID),
				zap.Error(err),
			)
		}
		return
	}

	start, end, err := parseRange(rangeHeader, file.Size)
	if err != nil {
		writeError(w, http.StatusRequestedRangeNotSatisfiable, err.Error())
		return
	}

	length := end - start + 1
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.Size))
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.WriteHeader(http.StatusPartialContent)

	if err := s.tg.DownloadRange(r.Context(), fileID, w, start, length); err != nil {
		// "connection reset by peer" is normal when the player seeks or cancels
		if r.Context().Err() != nil {
			s.logger.Debug("stream cancelled (client disconnected)",
				zap.String("file_id", fileID),
				zap.Int64("start", start),
			)
		} else {
			s.logger.Error("stream failed",
				zap.String("file_id", fileID),
				zap.Int64("start", start),
				zap.Int64("length", length),
				zap.Error(err),
			)
		}
	}
}

func (s *Server) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")

	thumb, err := s.tg.Storage().GetFileThumbnail(fileID)
	if err != nil || len(thumb) == 0 {
		writeError(w, http.StatusNotFound, "no thumbnail")
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(thumb)
}

func (s *Server) handlePlayer(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")

	file, err := s.tg.Storage().GetFile(fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	streamURL := fmt.Sprintf("/api/files/%s/stream", fileID)
	var mediaTag string
	if strings.HasPrefix(file.MimeType, "video/") {
		mediaTag = fmt.Sprintf(`<video controls autoplay preload="auto" src="%s" style="max-width:100vw;max-height:100vh;"></video>`, streamURL)
	} else if strings.HasPrefix(file.MimeType, "audio/") {
		mediaTag = fmt.Sprintf(`<audio controls autoplay src="%s"></audio>`, streamURL)
	} else if strings.HasPrefix(file.MimeType, "image/") {
		mediaTag = fmt.Sprintf(`<img src="%s" alt="%s" style="max-width:100vw;max-height:100vh;object-fit:contain;" />`, streamURL, file.Name)
	}

	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>%s</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{background:#111;display:flex;align-items:center;justify-content:center;min-height:100vh;}</style>
</head><body>%s</body></html>`, file.Name, mediaTag)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func parseRange(rangeHeader string, totalSize int64) (start, end int64, err error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range header")
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.SplitN(rangeSpec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	if parts[0] == "" {
		suffixLen, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid suffix range")
		}
		start = totalSize - suffixLen
		end = totalSize - 1
	} else {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start range")
		}
		if parts[1] == "" {
			end = totalSize - 1
		} else {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid end range")
			}
		}
	}

	if start < 0 || start >= totalSize || end >= totalSize || start > end {
		return 0, 0, fmt.Errorf("range not satisfiable")
	}

	return start, end, nil
}
