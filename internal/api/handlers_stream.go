package api

import (
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

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", file.MimeType)

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// Full stream
		w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
		if err := s.tg.DownloadFile(r.Context(), fileID, w); err != nil {
			s.logger.Error("stream failed", zap.String("file_id", fileID), zap.Error(err))
		}
		return
	}

	// Parse Range header: "bytes=START-END"
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
		s.logger.Error("range stream failed",
			zap.String("file_id", fileID),
			zap.Int64("start", start),
			zap.Int64("length", length),
			zap.Error(err),
		)
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
		// Suffix range: -500 means last 500 bytes
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
